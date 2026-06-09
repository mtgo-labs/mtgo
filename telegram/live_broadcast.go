package telegram

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// BroadcastStream manages an RTMP live stream to a Telegram group call via ffmpeg.
// Create one with NewBroadcastStream, fetch RTMP credentials, then call Play to start
// streaming and Stop to terminate.
//
// Example (complete streaming workflow):
//
//	stream, err := client.NewBroadcastStream(chatID)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Fetch RTMP credentials from Telegram.
//	if err := stream.FetchRTMPURL(ctx); err != nil {
//		log.Fatal(err)
//	}
//
//	// Start streaming a local file.
//	if err := stream.Play("video.mp4"); err != nil {
//		log.Fatal(err)
//	}
//
//	// Wait, then stop.
//	time.Sleep(30 * time.Second)
//	if err := stream.Stop(); err != nil {
//		log.Fatal(err)
//	}
type BroadcastStream struct {
	chatID int64
	url    string
	key    string
	client *Client
	state  BroadcastState
	cfg    *BroadcastConfig
	src    broadcastSource

	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stderr     bytes.Buffer
	cancelFunc context.CancelFunc

	mu        sync.Mutex
	onError   func(int64, error)
	onEnd     func(int64)
	lastError error
}

// NewBroadcastStream creates a new BroadcastStream for the specified chat.
// An optional BroadcastConfig can be provided; otherwise DefaultBroadcastConfig is used.
//
// Example:
//
//	stream, err := client.NewBroadcastStream(chatID)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Or with a custom config:
//	cfg := &telegram.BroadcastConfig{Bitrate: "4000k", FrameRate: 60}
//	stream, err = client.NewBroadcastStream(chatID, cfg)
func (c *Client) NewBroadcastStream(chatID int64, cfg ...*BroadcastConfig) (*BroadcastStream, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, ErrFFmpegNotFound
	}

	config := DefaultBroadcastConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		config = cfg[0]
	}

	return &BroadcastStream{
		chatID: chatID,
		client: c,
		state:  BroadcastIdle,
		cfg:    config,
	}, nil
}

// FetchRTMPURL retrieves the RTMP URL and stream key from Telegram without revoking
// any previously issued credentials.
func (s *BroadcastStream) FetchRTMPURL(ctx context.Context) error {
	if err := s.client.ensureConnected(); err != nil {
		return err
	}

	peer, err := resolvePeer(s.client, s.chatID)
	if err != nil {
		return fmt.Errorf("fetch rtmp: resolve peer: %w", err)
	}

	rpc := s.client.Raw()
	info, err := rpc.PhoneGetGroupCallStreamRtmpURL(ctx, &tg.PhoneGetGroupCallStreamRtmpURLRequest{
		Peer:   peer,
		Revoke: false,
	})
	if err != nil {
		return fmt.Errorf("fetch rtmp: %w", err)
	}

	s.mu.Lock()
	s.url = info.URL
	s.key = info.Key
	s.mu.Unlock()
	return nil
}

// RefreshRTMPURL revokes the existing RTMP credentials and fetches a new URL and key.
func (s *BroadcastStream) RefreshRTMPURL(ctx context.Context) error {
	if err := s.client.ensureConnected(); err != nil {
		return err
	}

	peer, err := resolvePeer(s.client, s.chatID)
	if err != nil {
		return fmt.Errorf("refresh rtmp: resolve peer: %w", err)
	}

	rpc := s.client.Raw()
	info, err := rpc.PhoneGetGroupCallStreamRtmpURL(ctx, &tg.PhoneGetGroupCallStreamRtmpURLRequest{
		Peer:   peer,
		Revoke: true,
	})
	if err != nil {
		return fmt.Errorf("refresh rtmp: %w", err)
	}

	s.mu.Lock()
	s.url = info.URL
	s.key = info.Key
	s.mu.Unlock()
	return nil
}

// SetURL sets the RTMP endpoint URL directly, bypassing Telegram's RTMP API.
func (s *BroadcastStream) SetURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.url = url
}

// SetKey sets the RTMP stream key directly.
func (s *BroadcastStream) SetKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key = key
}

// SetFullURL parses a complete RTMP URL (rtmp://host/s/key) and sets both the URL and key.
func (s *BroadcastStream) SetFullURL(fullURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fullURL == "" {
		return ErrRTMPURLRequired
	}
	if !strings.HasPrefix(fullURL, "rtmp://") && !strings.HasPrefix(fullURL, "rtmps://") {
		return ErrRTMPURLInvalid
	}

	parts := strings.SplitN(fullURL, "/s/", 2)
	if len(parts) == 2 {
		s.url = parts[0] + "/s/"
		s.key = parts[1]
		return nil
	}

	idx := strings.LastIndex(fullURL, "/")
	if idx < 0 {
		return ErrRTMPURLBadFormat
	}
	s.url = fullURL[:idx+1]
	s.key = fullURL[idx+1:]
	return nil
}

// URL returns the current RTMP endpoint URL.
func (s *BroadcastStream) URL() string {
	return s.url
}

// Key returns the current RTMP stream key.
func (s *BroadcastStream) Key() string {
	return s.key
}

// FullURL returns the complete RTMP URL by concatenating the URL and key.
func (s *BroadcastStream) FullURL() string {
	return s.url + s.key
}

// State returns the current broadcast state.
func (s *BroadcastStream) State() BroadcastState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// LastError returns the most recent error encountered during streaming, if any.
func (s *BroadcastStream) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastError
}

// OnError registers a callback invoked when the stream encounters an error.
func (s *BroadcastStream) OnError(fn func(int64, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onError = fn
}

// OnEnd registers a callback invoked when the stream ends normally.
func (s *BroadcastStream) OnEnd(fn func(int64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onEnd = fn
}

// Play starts streaming the given source (a file path string or []byte) to the
// RTMP endpoint. Returns an error if already playing or no RTMP URL is configured.
//
// Example (stream from a file):
//
//	if err := stream.Play("video.mp4"); err != nil {
//		log.Fatal(err)
//	}
//
// Example (stream from a byte slice):
//
//	data, _ := os.ReadFile("clip.mp4")
//	if err := stream.Play(data); err != nil {
//		log.Fatal(err)
//	}
func (s *BroadcastStream) Play(source interface{}) error {
	if s.url == "" || s.key == "" {
		return ErrNoRTMPURL
	}

	s.mu.Lock()
	if s.state == BroadcastPlaying {
		s.mu.Unlock()
		return ErrAlreadyPlaying
	}

	switch src := source.(type) {
	case string:
		if _, err := os.Stat(src); err != nil {
			s.mu.Unlock()
			if os.IsNotExist(err) {
				return fmt.Errorf("%w: %s", ErrFileNotFound, src)
			}
			return fmt.Errorf("play: access file: %w", err)
		}
		s.src.filePath = src
		s.src.data = nil
		s.mu.Unlock()
		return s.startFFmpeg(src, false)

	case []byte:
		if len(src) == 0 {
			s.mu.Unlock()
			return ErrPlayEmptyInput
		}
		s.src.data = src
		s.src.filePath = ""
		s.mu.Unlock()
		return s.startFFmpeg("pipe:0", true)

	default:
		s.mu.Unlock()
		return fmt.Errorf("play: unsupported source type %T", source)
	}
}

// PlayAudioWithImage streams an audio source combined with a static image as video.
func (s *BroadcastStream) PlayAudioWithImage(audioSource interface{}, imageFile string) error {
	if s.url == "" || s.key == "" {
		return ErrNoRTMPURL
	}

	if _, err := os.Stat(imageFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, imageFile)
		}
		return fmt.Errorf("play audio with image: access image: %w", err)
	}

	s.mu.Lock()
	if s.state == BroadcastPlaying {
		s.mu.Unlock()
		return ErrAlreadyPlaying
	}

	s.src.audioOnly = true
	s.src.image = imageFile

	switch src := audioSource.(type) {
	case string:
		if _, err := os.Stat(src); err != nil {
			s.mu.Unlock()
			if os.IsNotExist(err) {
				return fmt.Errorf("%w: %s", ErrFileNotFound, src)
			}
			return fmt.Errorf("play audio with image: access audio: %w", err)
		}
		s.src.filePath = src
		s.src.data = nil
		s.mu.Unlock()
		return s.startFFmpeg(src, false)

	case []byte:
		if len(src) == 0 {
			s.mu.Unlock()
			return ErrPlayAudioEmptyInput
		}
		s.src.data = src
		s.src.filePath = ""
		s.mu.Unlock()
		return s.startFFmpeg("pipe:0", true)

	default:
		s.mu.Unlock()
		return fmt.Errorf("play audio with image: unsupported source type %T", audioSource)
	}
}

// Pause stops the ffmpeg process and records the current playback position for later resume.
func (s *BroadcastStream) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != BroadcastPlaying {
		return fmt.Errorf("pause: stream is %s", s.state)
	}

	s.src.pausedAt = time.Since(s.src.startTime) + s.src.seekPos
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.state = BroadcastPaused
	return nil
}

// Resume restarts streaming from the paused position.
func (s *BroadcastStream) Resume() error {
	s.mu.Lock()
	if s.state != BroadcastPaused {
		s.mu.Unlock()
		return ErrNotPaused
	}
	s.src.seekPos = s.src.pausedAt
	s.mu.Unlock()

	if s.src.filePath != "" {
		return s.startFFmpeg(s.src.filePath, false)
	} else if s.src.data != nil {
		return s.startFFmpeg("pipe:0", true)
	}
	return ErrNoInputSource
}

// Seek moves the playback position. Only supported for file-based inputs.
func (s *BroadcastStream) Seek(position time.Duration) error {
	s.mu.Lock()
	if s.src.filePath == "" {
		s.mu.Unlock()
		return ErrSeekFileOnly
	}
	if position < 0 {
		s.mu.Unlock()
		return ErrSeekNegativePos
	}
	wasPlaying := s.state == BroadcastPlaying
	s.src.seekPos = position
	s.mu.Unlock()

	if wasPlaying {
		_ = s.Stop()
		return s.startFFmpeg(s.src.filePath, false)
	}
	return nil
}

// CurrentPosition returns the elapsed playback duration based on the stream state.
func (s *BroadcastStream) CurrentPosition() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.state {
	case BroadcastPlaying:
		return time.Since(s.src.startTime) + s.src.seekPos
	case BroadcastPaused:
		return s.src.pausedAt
	default:
		return 0
	}
}

// Mute disables audio output by restarting ffmpeg with volume set to zero.
func (s *BroadcastStream) Mute() error {
	s.mu.Lock()
	if s.src.muted {
		s.mu.Unlock()
		return nil
	}
	if s.state != BroadcastPlaying {
		s.mu.Unlock()
		return fmt.Errorf("mute: stream is %s", s.state)
	}

	s.src.muted = true
	pos := time.Since(s.src.startTime) + s.src.seekPos
	s.mu.Unlock()

	_ = s.Stop()
	s.mu.Lock()
	s.src.seekPos = pos
	s.mu.Unlock()

	if s.src.filePath != "" {
		return s.startFFmpeg(s.src.filePath, false)
	} else if s.src.data != nil {
		return s.startFFmpeg("pipe:0", true)
	}
	return nil
}

// Unmute re-enables audio output by restarting ffmpeg without the mute filter.
func (s *BroadcastStream) Unmute() error {
	s.mu.Lock()
	if !s.src.muted {
		s.mu.Unlock()
		return nil
	}
	if s.state != BroadcastPlaying {
		s.mu.Unlock()
		return fmt.Errorf("unmute: stream is %s", s.state)
	}

	s.src.muted = false
	pos := time.Since(s.src.startTime) + s.src.seekPos
	s.mu.Unlock()

	_ = s.Stop()
	s.mu.Lock()
	s.src.seekPos = pos
	s.mu.Unlock()

	if s.src.filePath != "" {
		return s.startFFmpeg(s.src.filePath, false)
	} else if s.src.data != nil {
		return s.startFFmpeg("pipe:0", true)
	}
	return nil
}

// IsMuted reports whether the stream audio is currently muted.
func (s *BroadcastStream) IsMuted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.src.muted
}

// Stop terminates the ffmpeg process and resets the stream state.
//
// Example:
//
//	if err := stream.Stop(); err != nil {
//		log.Printf("stop error: %v", err)
//	}
//	fmt.Println("stream state:", stream.State())
//	// Output: stream state: stopped
func (s *BroadcastStream) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == BroadcastIdle || s.state == BroadcastStopped {
		return nil
	}

	s.state = BroadcastStopped
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	return nil
}

// SetBitrate updates the video bitrate for subsequent playback.
func (s *BroadcastStream) SetBitrate(bitrate string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Bitrate = bitrate
}

// SetAudioBitrate updates the audio bitrate for subsequent playback.
func (s *BroadcastStream) SetAudioBitrate(bitrate string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.AudioBit = bitrate
}

// SetFrameRate updates the output frame rate in fps for subsequent playback.
func (s *BroadcastStream) SetFrameRate(fps int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.FrameRate = fps
}

// SetLoopCount sets the number of times the input source should loop (0 = no loop).
func (s *BroadcastStream) SetLoopCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.LoopCount = count
}

func (s *BroadcastStream) startFFmpeg(input string, pipeInput bool) error {
	ctx, cancel := context.WithCancel(context.Background())

	args := s.buildFFmpegArgs(input)
	s.cmd = exec.CommandContext(ctx, "ffmpeg", args...)

	s.stderr.Reset()
	s.cmd.Stderr = &s.stderr
	s.cmd.Stdout = nil

	if pipeInput {
		stdin, err := s.cmd.StdinPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("ffmpeg: stdin pipe: %w", err)
		}
		s.stdin = stdin
	}

	s.cancelFunc = cancel

	if err := s.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("ffmpeg: start: %w", err)
	}

	s.mu.Lock()
	s.state = BroadcastPlaying
	s.src.startTime = time.Now()
	s.lastError = nil
	s.mu.Unlock()

	if pipeInput && s.src.data != nil {
		go func() {
			defer s.stdin.Close()
			defer func() {
				if r := recover(); r != nil {
					s.client.Log.Errorf("broadcast stdin write panic: %v", r)
				}
			}()
			s.stdin.Write(s.src.data)
		}()
	}

	go s.watchProcess(ctx)
	return nil
}

func (s *BroadcastStream) watchProcess(ctx context.Context) {
	err := s.cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != BroadcastPlaying {
		return
	}

	s.state = BroadcastIdle
	if err != nil && ctx.Err() == nil {
		errMsg := s.stderr.String()
		if errMsg != "" {
			s.lastError = fmt.Errorf("ffmpeg: %s", errMsg)
		} else {
			s.lastError = fmt.Errorf("ffmpeg: %w", err)
		}
		if s.onError != nil {
			go s.onError(s.chatID, s.lastError)
		}
	} else {
		if s.onEnd != nil {
			go s.onEnd(s.chatID)
		}
	}
}

func (s *BroadcastStream) buildFFmpegArgs(input string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var args []string

	if s.src.seekPos > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", s.src.seekPos.Seconds()))
	}

	args = append(args, "-re")

	if s.src.audioOnly && s.src.image != "" {
		args = append(
			args,
			"-loop", "1",
			"-i", s.src.image,
		)
		if s.cfg.LoopCount != 0 {
			args = append(args, "-stream_loop", fmt.Sprintf("%d", s.cfg.LoopCount))
		}
		args = append(args, "-i", input)
		args = append(
			args,
			"-map", "0:v",
			"-map", "1:a",
			"-c:v", "libx264",
			"-preset", "superfast",
			"-b:v", s.cfg.Bitrate,
			"-maxrate", s.cfg.Bitrate,
			"-bufsize", doubleBitrate(s.cfg.Bitrate),
			"-pix_fmt", "yuv420p",
			"-r", fmt.Sprintf("%d", s.cfg.FrameRate),
			"-g", fmt.Sprintf("%d", s.cfg.FrameRate),
			"-threads", "0",
		)
		args = append(args, s.audioArgs()...)
		args = append(
			args,
			"-shortest",
			"-f", "flv",
			"-rtmp_buffer", "100",
			"-rtmp_live", "live",
			s.url+s.key,
		)
	} else {
		if s.cfg.LoopCount != 0 {
			args = append(args, "-stream_loop", fmt.Sprintf("%d", s.cfg.LoopCount))
		}
		args = append(
			args,
			"-i", input,
			"-c:v", "libx264",
			"-preset", "superfast",
			"-b:v", s.cfg.Bitrate,
			"-maxrate", s.cfg.Bitrate,
			"-bufsize", doubleBitrate(s.cfg.Bitrate),
			"-pix_fmt", "yuv420p",
			"-g", fmt.Sprintf("%d", s.cfg.FrameRate),
			"-threads", "0",
		)
		args = append(args, s.audioArgs()...)
		args = append(
			args,
			"-f", "flv",
			"-rtmp_buffer", "100",
			"-rtmp_live", "live",
			s.url+s.key,
		)
	}

	return args
}

func (s *BroadcastStream) audioArgs() []string {
	args := []string{
		"-c:a", "aac",
		"-b:a", s.cfg.AudioBit,
		"-ac", "2",
		"-ar", "44100",
	}
	if s.src.muted {
		args = append(args, "-af", "volume=0")
	}
	return args
}

func doubleBitrate(bitrate string) string {
	var val int
	fmt.Sscanf(bitrate, "%dk", &val)
	return fmt.Sprintf("%dk", val*2)
}
