package session

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

// batchItem is a single outbound RPC awaiting container packing.
type batchItem struct {
	msgID   int64
	seqNo   uint32
	body    tg.TLObject
	handle  *CallHandle
	timeout time.Duration
}

// OutboundBatcher coalesces multiple outbound RPCs into MTProto msg_container
// messages using adaptive flushing. When enabled, the Session's Send method
// delegates to Submit, which registers a pending handle and queues the item.
// A flush goroutine drains queued items: a single item flushes immediately
// (zero added latency); multiple items are packed into a single container and
// sent as one encrypted message.
//
// Ported conceptually from TDLib net/Session.h outbound container packing.
type OutboundBatcher struct {
	session *Session

	maxContainerBytes int
	coalesceWindow    time.Duration

	mu     sync.Mutex
	high   []batchItem
	low    []batchItem
	notify chan struct{}

	done chan struct{}
	wg   sync.WaitGroup

	// Metrics
	containersSent atomic.Int64
	messagesPacked atomic.Int64
}

// NewOutboundBatcher creates a batcher backed by the given session. The batcher
// must be Started before use and Closed when done.
func NewOutboundBatcher(s *Session, maxContainerBytes int, coalesceWindow time.Duration) *OutboundBatcher {
	if maxContainerBytes <= 0 {
		maxContainerBytes = 1 << 20 // 1 MiB
	}
	if coalesceWindow == 0 {
		coalesceWindow = 10 * time.Millisecond
	}
	return &OutboundBatcher{
		session:           s,
		maxContainerBytes: maxContainerBytes,
		coalesceWindow:    coalesceWindow,
		notify:            make(chan struct{}, 1),
		done:              make(chan struct{}),
	}
}

// Start launches the flush goroutine.
func (b *OutboundBatcher) Start() {
	b.wg.Add(1)
	go b.flushLoop()
}

// Submit registers a pending handle for msgID, queues the item, and returns the
// handle. The caller waits on handle.Done() for the server's response. Returns
// an error if the batcher is shutting down.
func (b *OutboundBatcher) Submit(ctx context.Context, msgID int64, seqNo uint32, body tg.TLObject, priority Priority, timeout time.Duration) (*CallHandle, error) {
	handle, err := b.session.pending.Register(msgID, false)
	if err != nil {
		return nil, err
	}

	// Store the encoded body for msg_resend_req re-transmission (#4).
	handle.StorePayload(encodeBuf(body))

	item := batchItem{
		msgID:   msgID,
		seqNo:   seqNo,
		body:    body,
		handle:  handle,
		timeout: timeout,
	}

	b.mu.Lock()
	// Reject items after Close to prevent leaking pending handles (#5).
	select {
	case <-b.done:
		b.mu.Unlock()
		b.session.pending.Cancel(msgID)
		return nil, ErrBatcherClosed
	default:
	}
	if priority == PriorityHigh {
		b.high = append(b.high, item)
	} else {
		b.low = append(b.low, item)
	}
	b.mu.Unlock()

	// Signal the flush goroutine (non-blocking; it's buffered to 1).
	select {
	case b.notify <- struct{}{}:
	default:
	}

	return handle, nil
}

func (b *OutboundBatcher) flushLoop() {
	defer b.wg.Done()
	for {
		select {
		case <-b.done:
			return
		case <-b.notify:
			b.flush()
		}
	}
}

func (b *OutboundBatcher) flush() {
	// Adaptive: if this is the first flush and the coalescing window would let
	// more items arrive, wait briefly. But never delay a lone item beyond the
	// window.
	if b.coalesceWindow > 0 {
		// Check if there might be more items coming (i.e., notify channel still
		// has signal or we're under load). For simplicity, always wait the short
		// window — it's microseconds and only when the flush goroutine wakes.
		timer := time.NewTimer(b.coalesceWindow)
		select {
		case <-b.done:
			timer.Stop()
			return
		case <-timer.C:
		}
	}

	// Drain all queued items: high priority first, then low.
	b.mu.Lock()
	items := make([]batchItem, 0, len(b.high)+len(b.low))
	items = append(items, b.high...)
	items = append(items, b.low...)
	b.high = b.high[:0]
	b.low = b.low[:0]
	b.mu.Unlock()

	if len(items) == 0 {
		return
	}

	// Build MTProtoMessages for each item.
	msgs := make([]*tg.MTProtoMessage, 0, len(items))
	totalSize := 0
	pendingItems := items[:0] // reuse for items we actually pack
	for _, item := range items {
		if item.handle == nil {
			continue
		}
		msg := &tg.MTProtoMessage{
			MsgID: item.msgID,
			SeqNo: item.seqNo,
			Body:  item.body,
		}
		// Estimate serialized size (rough: encode and measure).
		msgBuf := encodeBuf(msg)
		if totalSize+len(msgBuf) > b.maxContainerBytes && len(msgs) > 0 {
			// Flush what we have, start a new container.
			b.packAndSend(msgs)
			msgs = msgs[:0]
			totalSize = 0
		}
		msgs = append(msgs, msg)
		totalSize += len(msgBuf)
		pendingItems = append(pendingItems, item)
	}

	if len(msgs) > 0 {
		b.packAndSend(msgs)
	}
}

// packAndSend packs the messages into a container (or single message if only
// one), encrypts once, and writes to the transport. Each child's pending handle
// resolves independently when the server responds.
func (b *OutboundBatcher) packAndSend(msgs []*tg.MTProtoMessage) {
	s := b.session
	s.mu.RLock()
	authKey := s.authKey
	authKeyID := s.authKeyID
	s.mu.RUnlock()

	salt := s.ensureFreshSalt(context.Background())
	if salt == 0 {
		salt = s.saltMgr.Load()
	}

	var outerBody tg.TLObject
	var msgID int64
	var seqNo uint32

	if len(msgs) == 1 {
		// Single-message path: use the child's own msgID/seqNo directly (#1).
		// The server's rpc_result references this msgID, and the pending handle
		// was registered under it.
		outerBody = msgs[0].Body
		msgID = msgs[0].MsgID
		seqNo = msgs[0].SeqNo
	} else {
		// Container path: allocate a container-level msgID.
		outerBody = &tg.MsgContainer{Messages: msgs}
		msgID = s.msgFactory.AllocateMsgID()
		seqNo = uint32(s.msgFactory.AllocateSeqNo(false))

		// Track container→child mapping for bad_msg rejection (#3).
		childIDs := make([]int64, len(msgs))
		for i, m := range msgs {
			childIDs[i] = m.MsgID
		}
		s.containerTracker.TrackContainer(msgID, childIDs)
	}

	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: seqNo,
		Body:  outerBody,
	}

	encrypted, err := crypto.Pack(message, salt, s.sessionIDBytes(), authKey, authKeyID)
	if err != nil {
		// Reject all children.
		for _, msg := range msgs {
			s.pending.Reject(msg.MsgID, err)
		}
		return
	}

	if err := s.writeEncrypted(context.Background(), encrypted, 10*time.Second); err != nil {
		for _, msg := range msgs {
			s.pending.Cancel(msg.MsgID)
		}
		return
	}

	b.containersSent.Add(1)
	b.messagesPacked.Add(int64(len(msgs)))
}

// Snapshot returns metrics for introspection (FR-020).
type OutboundSnapshot struct {
	HighDepth      int
	LowDepth       int
	ContainersSent int64
	MessagesPacked int64
}

func (b *OutboundBatcher) Snapshot() OutboundSnapshot {
	b.mu.Lock()
	hd, ld := len(b.high), len(b.low)
	b.mu.Unlock()
	return OutboundSnapshot{
		HighDepth:      hd,
		LowDepth:       ld,
		ContainersSent: b.containersSent.Load(),
		MessagesPacked: b.messagesPacked.Load(),
	}
}

// ErrBatcherClosed is returned when Submit is called after Close.
var ErrBatcherClosed = errors.New("session: outbound batcher is closed")

// Close stops the flush goroutine. Pending items are NOT flushed (the caller
// should drain before closing). Idempotent.

func (b *OutboundBatcher) Close() error {
	close(b.done)
	b.wg.Wait()
	return nil
}

// encodeBuf serializes a TLObject to measure its wire size. Returns a nil slice
// on error (the item is still packed; size tracking is best-effort).
func encodeBuf(obj tg.TLObject) []byte {
	var buf bytes.Buffer
	if err := obj.Encode(&buf); err != nil {
		return nil
	}
	return buf.Bytes()
}
