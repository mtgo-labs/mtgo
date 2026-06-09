package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
)

func TestProgressInfo(t *testing.T) {
	info := params.ProgressInfo{
		FileName:      "photo.jpg",
		TotalBytes:    1048576,
		UploadedBytes: 524288,
		IsUpload:      true,
	}

	if info.Progress() != 50.0 {
		t.Errorf("Progress() = %f, want 50.0", info.Progress())
	}

	info.UploadedBytes = 1048576
	if info.Progress() != 100.0 {
		t.Errorf("Progress() = %f, want 100.0", info.Progress())
	}
}

func TestProgressInfoZeroTotal(t *testing.T) {
	info := params.ProgressInfo{
		FileName:   "empty.txt",
		TotalBytes: 0,
	}
	if info.Progress() != 0 {
		t.Errorf("Progress() with zero total = %f, want 0", info.Progress())
	}
}

func TestProgressInfoDownload(t *testing.T) {
	info := params.ProgressInfo{
		FileName:        "video.mp4",
		TotalBytes:      10485760,
		DownloadedBytes: 2097152,
		IsUpload:        false,
	}
	if info.Progress() != 20.0 {
		t.Errorf("Progress() = %f, want 20.0", info.Progress())
	}
}

func TestUploadConstants(t *testing.T) {
	if uploadPartSize != 524288 {
		t.Errorf("uploadPartSize = %d, want 524288", uploadPartSize)
	}
	if bigFileThreshold != 10*1024*1024 {
		t.Errorf("bigFileThreshold = %d, want %d", bigFileThreshold, 10*1024*1024)
	}
	if downloadChunkSize != 1048576 {
		t.Errorf("downloadChunkSize = %d, want 1048576", downloadChunkSize)
	}
	if maxFileSize != 2*1024*1024*1024 {
		t.Errorf("maxFileSize = %d, want %d", maxFileSize, 2*1024*1024*1024)
	}
}

func TestUploadOptionsDefaults(t *testing.T) {
	opts := &UploadOptions{}
	if opts.Workers != 0 {
		t.Errorf("default Workers = %d, want 0", opts.Workers)
	}
	if opts.Progress != nil {
		t.Error("default Progress should be nil")
	}
}

func TestDownloadDefaults(t *testing.T) {
	opts := &params.Download{}
	if opts.ChunkSize != 0 {
		t.Errorf("default ChunkSize = %d, want 0", opts.ChunkSize)
	}
	if opts.Progress != nil {
		t.Error("default Progress should be nil")
	}
}
