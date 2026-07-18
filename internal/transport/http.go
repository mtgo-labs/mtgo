package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultHTTPMaxWait       = 25_000
	defaultHTTPMaxInFlight   = 16
	maxHTTPInFlight          = 1024
	defaultHTTPInboxSize     = 32
	defaultHTTPRetryInterval = 500 * time.Millisecond
	defaultHTTPRequestMargin = 5 * time.Second
)

// HTTPConfig configures MTProto's HTTP transport. URLs must be complete /api
// endpoints. MaxDelay, WaitAfter, and MaxWait are MTProto http_wait values in
// milliseconds.
type HTTPConfig struct {
	URLs             []string
	Client           *http.Client
	MaxDelay         int32
	WaitAfter        int32
	MaxWait          int32
	MaxInFlight      int
	MaxResponseBytes int64
	RetryInterval    time.Duration
	CloseIdleConns   bool
	OnRequest        func(endpoint string, latency time.Duration, err error)
}

type httpResult struct {
	data []byte
	err  error
}

// HTTP implements the MTProto HTTP transport. Send submits bounded
// asynchronous POSTs while Recv returns non-empty response bodies.
type HTTP struct {
	client           *http.Client
	urls             []string
	maxDelay         int32
	waitAfter        int32
	maxWait          int32
	maxResponseBytes int64
	retryInterval    time.Duration
	requestTimeout   time.Duration
	closeIdleConns   bool
	onRequest        func(endpoint string, latency time.Duration, err error)

	ctx    context.Context
	cancel context.CancelFunc
	inbox  chan httpResult
	slots  chan struct{}

	nextURL       atomic.Uint32
	closed        atomic.Bool
	readDeadline  atomic.Int64
	writeDeadline atomic.Int64

	lifecycle sync.Mutex
	wg        sync.WaitGroup
	waitOnce  sync.Once
}

// NewHTTP creates an MTProto HTTP transport. It does not perform network I/O.
func NewHTTP(cfg HTTPConfig) (*HTTP, error) {
	if len(cfg.URLs) == 0 {
		return nil, errors.New("http transport: at least one URL is required")
	}
	urls := make([]string, len(cfg.URLs))
	for i, rawURL := range cfg.URLs {
		u, err := url.Parse(rawURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, fmt.Errorf("http transport: invalid URL %q", rawURL)
		}
		urls[i] = u.String()
	}

	maxWait := cfg.MaxWait
	if maxWait <= 0 {
		maxWait = defaultHTTPMaxWait
	}
	maxInFlight := cfg.MaxInFlight
	if maxInFlight <= 0 {
		maxInFlight = defaultHTTPMaxInFlight
	}
	if maxInFlight > maxHTTPInFlight {
		return nil, fmt.Errorf("http transport: MaxInFlight %d exceeds limit %d", maxInFlight, maxHTTPInFlight)
	}
	maxResponseBytes := cfg.MaxResponseBytes
	if maxResponseBytes <= 0 || maxResponseBytes > MaxPayloadLen {
		maxResponseBytes = MaxPayloadLen
	}
	retryInterval := cfg.RetryInterval
	if retryInterval <= 0 {
		retryInterval = defaultHTTPRetryInterval
	}
	client := cfg.Client
	closeIdleConns := cfg.CloseIdleConns
	if client == nil {
		client = &http.Client{Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConns:        maxInFlight,
			MaxIdleConnsPerHost: maxInFlight,
			MaxConnsPerHost:     maxInFlight,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
		}, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		closeIdleConns = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &HTTP{
		client:           client,
		urls:             urls,
		maxDelay:         cfg.MaxDelay,
		waitAfter:        cfg.WaitAfter,
		maxWait:          maxWait,
		maxResponseBytes: maxResponseBytes,
		retryInterval:    retryInterval,
		requestTimeout:   time.Duration(maxWait)*time.Millisecond + defaultHTTPRequestMargin,
		closeIdleConns:   closeIdleConns,
		onRequest:        cfg.OnRequest,
		ctx:              ctx,
		cancel:           cancel,
		inbox:            make(chan httpResult, defaultHTTPInboxSize),
		slots:            make(chan struct{}, maxInFlight),
	}, nil
}

// Send submits one encrypted MTProto payload. The payload is copied because
// callers may reuse their buffer after Send returns.
func (h *HTTP) Send(buf *bytes.Buffer) error {
	if h.closed.Load() {
		return net.ErrClosed
	}
	if buf.Len() > MaxPayloadLen {
		return ErrPayloadTooLarge
	}
	ctx, cancel := h.requestContext(h.writeDeadline.Load())
	select {
	case h.slots <- struct{}{}:
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}

	payload := bytes.Clone(buf.Bytes())
	h.lifecycle.Lock()
	if h.closed.Load() {
		h.lifecycle.Unlock()
		<-h.slots
		cancel()
		return net.ErrClosed
	}
	h.wg.Add(1)
	h.lifecycle.Unlock()

	go func() {
		defer h.wg.Done()
		defer func() { <-h.slots }()
		defer cancel()
		data, err := h.post(ctx, payload)
		if err != nil {
			h.rotateURL()
			h.deliver(httpResult{err: err})
			return
		}
		if len(data) > 0 {
			h.deliver(httpResult{data: data})
		}
	}()
	return nil
}

// Recv blocks until an HTTP response contains an MTProto payload.
func (h *HTTP) Recv() ([]byte, error) {
	ctx, cancel := h.requestContext(h.readDeadline.Load())
	defer cancel()
	select {
	case result := <-h.inbox:
		return result.data, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (h *HTTP) Close() error {
	h.lifecycle.Lock()
	if !h.closed.CompareAndSwap(false, true) {
		h.lifecycle.Unlock()
		return nil
	}
	h.cancel()
	h.lifecycle.Unlock()
	h.wg.Wait()
	if h.closeIdleConns {
		h.client.CloseIdleConnections()
	}
	return nil
}

func (h *HTTP) IsConnected() bool {
	return !h.closed.Load()
}

func (h *HTTP) SetWriteDeadline(deadline time.Time) error {
	h.writeDeadline.Store(deadlineUnixNano(deadline))
	return nil
}

func (h *HTTP) SetReadDeadline(deadline time.Time) error {
	h.readDeadline.Store(deadlineUnixNano(deadline))
	return nil
}

// HTTPWaitParams returns the values used for encrypted http_wait messages.
func (h *HTTP) HTTPWaitParams() (maxDelay, waitAfter, maxWait int32) {
	return h.maxDelay, h.waitAfter, h.maxWait
}

// StartHTTPWait starts exactly one long-poll loop. frame must return a newly
// encrypted http_wait message for every request.
func (h *HTTP) StartHTTPWait(frame func(context.Context) ([]byte, error)) {
	h.waitOnce.Do(func() {
		h.lifecycle.Lock()
		if h.closed.Load() {
			h.lifecycle.Unlock()
			return
		}
		h.wg.Add(1)
		h.lifecycle.Unlock()
		go func() {
			defer h.wg.Done()
			h.poll(frame)
		}()
	})
}

func (h *HTTP) poll(frame func(context.Context) ([]byte, error)) {
	for {
		payload, err := frame(h.ctx)
		if err != nil {
			h.deliver(httpResult{err: fmt.Errorf("http transport: build http_wait: %w", err)})
			return
		}
		started := time.Now()
		ctx, cancel := context.WithTimeout(h.ctx, h.requestTimeout)
		select {
		case h.slots <- struct{}{}:
		case <-ctx.Done():
			cancel()
			if h.ctx.Err() != nil {
				return
			}
			if !h.sleep(h.retryInterval) {
				return
			}
			continue
		}
		data, err := h.post(ctx, payload)
		<-h.slots
		cancel()
		if err != nil {
			if h.ctx.Err() != nil {
				return
			}
			if IsTransportError(err) || errors.Is(err, ErrPayloadTooLarge) {
				h.deliver(httpResult{err: err})
				return
			}
			h.rotateURL()
			if !h.sleep(h.retryInterval) {
				return
			}
			continue
		}
		if len(data) > 0 && !h.deliver(httpResult{data: data}) {
			return
		}
		if len(data) == 0 && time.Since(started) < h.retryInterval && !h.sleep(h.retryInterval) {
			return
		}
	}
}

func (h *HTTP) post(ctx context.Context, payload []byte) (data []byte, err error) {
	endpoint := h.currentURL()
	started := time.Now()
	if h.onRequest != nil {
		defer func() {
			h.onRequest(endpoint, time.Since(started), err)
		}()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("http transport: create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/octet-stream")
	response, err := h.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http transport: POST: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("http transport: unexpected status %s", response.Status)
	}
	data, err = io.ReadAll(io.LimitReader(response.Body, h.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("http transport: read response: %w", err)
	}
	if int64(len(data)) > h.maxResponseBytes {
		return nil, ErrPayloadTooLarge
	}
	if transportErr := DetectTransportError(data); transportErr != nil {
		return nil, transportErr
	}
	return data, nil
}

func (h *HTTP) currentURL() string {
	return h.urls[int(h.nextURL.Load())%len(h.urls)]
}

func (h *HTTP) rotateURL() {
	if len(h.urls) > 1 {
		h.nextURL.Add(1)
	}
}

func (h *HTTP) deliver(result httpResult) bool {
	select {
	case h.inbox <- result:
		return true
	case <-h.ctx.Done():
		return false
	}
}

func (h *HTTP) sleep(delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-h.ctx.Done():
		return false
	}
}

func (h *HTTP) requestContext(deadlineNanos int64) (context.Context, context.CancelFunc) {
	deadline := time.Now().Add(h.requestTimeout)
	if deadlineNanos != 0 {
		configured := time.Unix(0, deadlineNanos)
		if configured.Before(deadline) {
			deadline = configured
		}
	}
	return context.WithDeadline(h.ctx, deadline)
}

func deadlineUnixNano(deadline time.Time) int64 {
	if deadline.IsZero() {
		return 0
	}
	return deadline.UnixNano()
}
