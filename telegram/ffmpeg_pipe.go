package telegram

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func (s *BroadcastStream) StartPipe() error {
	if s.url == "" || s.key == "" {
		return ErrNoRTMPURL
	}

	s.mu.Lock()
	if s.state == BroadcastPlaying {
		s.mu.Unlock()
		return ErrAlreadyPlaying
	}
	s.src.filePath = ""
	s.src.data = nil
	s.mu.Unlock()

	return s.startFFmpegPipe()
}

func (s *BroadcastStream) FeedChunk(data []byte) error {
	s.mu.Lock()
	if s.state != BroadcastPlaying {
		s.mu.Unlock()
		return fmt.Errorf("feed chunk: %w: %s", ErrStreamNotPlaying, s.state)
	}
	if s.stdin == nil {
		s.mu.Unlock()
		return ErrStdinNotInit
	}
	s.mu.Unlock()

	_, err := s.stdin.Write(data)
	if err != nil {
		return fmt.Errorf("feed chunk: write: %w", err)
	}
	return nil
}

func (s *BroadcastStream) FeedReader(r io.Reader) error {
	s.mu.Lock()
	if s.state != BroadcastPlaying {
		s.mu.Unlock()
		return fmt.Errorf("feed reader: %w: %s", ErrStreamNotPlaying, s.state)
	}
	if s.stdin == nil {
		s.mu.Unlock()
		return ErrStdinNotInit
	}
	s.mu.Unlock()

	_, err := io.Copy(s.stdin, r)
	if err != nil {
		return fmt.Errorf("feed reader: copy: %w", err)
	}
	return nil
}

func (s *BroadcastStream) ClosePipe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdin != nil {
		return s.stdin.Close()
	}
	return nil
}

func (s *BroadcastStream) startFFmpegPipe() error {
	ctx, cancel := context.WithCancel(context.Background())

	args := s.buildFFmpegPipeArgs()
	s.cmd = exec.CommandContext(ctx, "ffmpeg", args...)

	s.stderr.Reset()
	s.cmd.Stderr = &s.stderr
	s.cmd.Stdout = nil

	stdin, err := s.cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("ffmpeg pipe: stdin: %w", err)
	}
	s.stdin = stdin
	s.cancelFunc = cancel

	if err := s.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("ffmpeg pipe: start: %w", err)
	}

	s.mu.Lock()
	s.state = BroadcastPlaying
	s.src.startTime = time.Now()
	s.lastError = nil
	s.mu.Unlock()

	go s.watchProcess(ctx)
	return nil
}

func (s *BroadcastStream) buildFFmpegPipeArgs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return []string{
		"-re",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "superfast",
		"-b:v", s.cfg.Bitrate,
		"-maxrate", s.cfg.Bitrate,
		"-bufsize", doubleBitrate(s.cfg.Bitrate),
		"-pix_fmt", "yuv420p",
		"-g", fmt.Sprintf("%d", s.cfg.FrameRate),
		"-threads", "0",
		"-c:a", "aac",
		"-b:a", s.cfg.AudioBit,
		"-ac", "2",
		"-ar", "44100",
		"-f", "flv",
		"-rtmp_buffer", "100",
		"-rtmp_live", "live",
		s.url + s.key,
	}
}
