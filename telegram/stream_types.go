package telegram

import (
	"errors"
	"time"
)

// BroadcastState represents the current state of a livestream broadcast.
//
// Possible states:
//   - BroadcastIdle – no stream is active.
//   - BroadcastPlaying – ffmpeg is actively streaming to the RTMP endpoint.
//   - BroadcastPaused – stream was paused; playback position is saved.
//   - BroadcastStopped – stream was explicitly stopped.
//
// Example:
//
//	stream, _ := client.NewBroadcastStream(chatID)
//	// ... after Play() ...
//	if stream.State() == telegram.BroadcastPlaying {
//		fmt.Println("stream is live")
//	}
type BroadcastState int

const (
	BroadcastIdle BroadcastState = iota
	BroadcastPlaying
	BroadcastPaused
	BroadcastStopped
)

func (s BroadcastState) String() string {
	switch s {
	case BroadcastIdle:
		return "idle"
	case BroadcastPlaying:
		return "playing"
	case BroadcastPaused:
		return "paused"
	case BroadcastStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// Broadcast stream errors.
//
// These errors are returned by BroadcastStream methods when the stream state,
// input, or RTMP configuration is invalid.
//
// Example:
//
//	err := stream.Play(ctx, file)
//	if errors.Is(err, telegram.ErrFFmpegNotFound) {
//		log.Fatal("ffmpeg is required for broadcasting")
//	}
var (
	// ErrFFmpegNotFound is returned when the ffmpeg binary cannot be found in
	// the system PATH. Broadcasting requires ffmpeg for media encoding.
	ErrFFmpegNotFound = errors.New("ffmpeg not found in path")
	// ErrAlreadyPlaying is returned when Play is called on a stream that is
	// already in the BroadcastPlaying state.
	ErrAlreadyPlaying = errors.New("stream is already playing")
	// ErrNotPaused is returned when Resume is called on a stream that is not
	// in the BroadcastPaused state.
	ErrNotPaused = errors.New("stream is not paused")
	// ErrNoRTMPURL is returned when an operation requires an RTMP endpoint but
	// neither FetchRTMPURL nor SetURL/SetKey has been called yet.
	ErrNoRTMPURL = errors.New("rtmp url not set; call FetchRTMPURL or SetURL/SetKey first")
	// ErrNoInputSource is returned when Play is called but no input source
	// (file, data, or pipe) has been configured.
	ErrNoInputSource = errors.New("no input source available")
	// ErrFileNotFound is returned when the configured input file path does not
	// exist on the local filesystem.
	ErrFileNotFound = errors.New("input file not found")
	// ErrNoGroupCall is returned when attempting to start a broadcast but
	// there is no active group call in the target chat.
	ErrNoGroupCall = errors.New("no active group call")
	// ErrNotChannel is returned when the target peer is not a channel or chat
	// that supports group calls.
	ErrNotChannel = errors.New("peer is not a channel or chat")
	// ErrPlayEmptyInput is returned when Play is called with an empty byte
	// slice as input.
	ErrPlayEmptyInput = errors.New("play: empty byte input")
	// ErrPlayAudioEmptyInput is returned when PlayAudioWithImage is called
	// with an empty audio byte slice.
	ErrPlayAudioEmptyInput = errors.New("play audio with image: empty byte input")
	// ErrSeekFileOnly is returned when Seek is called on a non-file input
	// source (e.g. a byte slice or pipe). Seeking is only supported for file
	// inputs.
	ErrSeekFileOnly = errors.New("seek: only supported for file input")
	// ErrSeekNegativePos is returned when Seek is called with a negative
	// duration value.
	ErrSeekNegativePos = errors.New("seek: position cannot be negative")
	// ErrStdinNotInit is returned when WriteToPipe is called before
	// StartPipe has initialized the stdin writer.
	ErrStdinNotInit = errors.New("stdin not initialized; call StartPipe first")
	// ErrRTMPURLRequired is returned when SetURL is called with an empty URL
	// string.
	ErrRTMPURLRequired = errors.New("rtmp url cannot be empty")
	// ErrRTMPURLInvalid is returned when the RTMP URL does not start with
	// "rtmp://" or "rtmps://".
	ErrRTMPURLInvalid = errors.New("invalid rtmp url: must start with rtmp:// or rtmps://")
	// ErrRTMPURLBadFormat is returned when the RTMP URL cannot be parsed into
	// a valid URL structure.
	ErrRTMPURLBadFormat = errors.New("invalid rtmp url format")
)

// BroadcastConfig holds FFmpeg encoding parameters for a broadcast stream.
//
// Example:
//
//	cfg := &telegram.BroadcastConfig{
//		Bitrate:   "4000k",
//		AudioBit:  "128k",
//		FrameRate: 60,
//		LoopCount: 0,
//	}
//	stream, _ := client.NewBroadcastStream(chatID, cfg)
type BroadcastConfig struct {
	Bitrate   string
	AudioBit  string
	FrameRate int
	LoopCount int
}

// DefaultBroadcastConfig returns a BroadcastConfig with sensible default encoding settings.
//
// Example:
//
//	cfg := telegram.DefaultBroadcastConfig()
//	fmt.Printf("video=%s audio=%s fps=%d\n", cfg.Bitrate, cfg.AudioBit, cfg.FrameRate)
//	// Output: video=2000k audio=96k fps=30
func DefaultBroadcastConfig() *BroadcastConfig {
	return &BroadcastConfig{
		Bitrate:   "2000k",
		AudioBit:  "96k",
		FrameRate: 30,
		LoopCount: -1,
	}
}

type broadcastSource struct {
	filePath  string
	data      []byte
	image     string
	audioOnly bool
	seekPos   time.Duration
	pausedAt  time.Duration
	muted     bool
	startTime time.Time
}
