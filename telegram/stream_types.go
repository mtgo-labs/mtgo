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

var (
	ErrFFmpegNotFound  = errors.New("ffmpeg not found in PATH")
	ErrAlreadyPlaying  = errors.New("stream is already playing")
	ErrNotPaused       = errors.New("stream is not paused")
	ErrNoRTMPURL       = errors.New("rtmp url not set; call FetchRTMPURL or SetURL/SetKey first")
	ErrNoInputSource   = errors.New("no input source available")
	ErrFileNotFound    = errors.New("input file not found")
	ErrNoGroupCall     = errors.New("no active group call")
	ErrNotChannel      = errors.New("peer is not a channel or chat")
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
	Bitrate    string
	AudioBit   string
	FrameRate  int
	LoopCount  int
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
	filePath string
	data     []byte
	image    string
	audioOnly bool
	seekPos   time.Duration
	pausedAt  time.Duration
	muted     bool
	startTime time.Time
}
