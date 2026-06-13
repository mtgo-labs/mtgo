package telegram

import "testing"

func TestBroadcastStateString(t *testing.T) {
	cases := []struct {
		state BroadcastState
		want  string
	}{
		{BroadcastIdle, "idle"},
		{BroadcastPlaying, "playing"},
		{BroadcastPaused, "paused"},
		{BroadcastStopped, "stopped"},
		{BroadcastState(999), "unknown"},
		{BroadcastState(-1), "unknown"},
	}
	for _, c := range cases {
		if got := c.state.String(); got != c.want {
			t.Errorf("BroadcastState(%d).String() = %q, want %q", c.state, got, c.want)
		}
	}
}

func TestDefaultBroadcastConfig(t *testing.T) {
	cfg := DefaultBroadcastConfig()
	if cfg == nil {
		t.Fatal("DefaultBroadcastConfig() returned nil")
	}
	if cfg.Bitrate != "2000k" {
		t.Errorf("Bitrate = %q, want 2000k", cfg.Bitrate)
	}
	if cfg.AudioBit != "96k" {
		t.Errorf("AudioBit = %q, want 96k", cfg.AudioBit)
	}
	if cfg.FrameRate != 30 {
		t.Errorf("FrameRate = %d, want 30", cfg.FrameRate)
	}
	if cfg.LoopCount != -1 {
		t.Errorf("LoopCount = %d, want -1", cfg.LoopCount)
	}
}
