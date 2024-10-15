package streaming

import (
	"context"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// Streamer defines the interface for streaming operations.
type Streamer interface {
	StartStream(ctx context.Context) error
	StopStream() error
	IsStreaming() bool
}

// FFmpegStreamer implements the Streamer interface using FFmpeg.
type FFmpegStreamer struct {
	cmd    *exec.Cmd
	mutex  sync.RWMutex
	status bool
	hlsDir string
	logger *logrus.Entry
}

// NewFFmpegStreamer creates a new FFmpegStreamer instance.
func NewFFmpegStreamer(hlsDir string, logger *logrus.Entry) *FFmpegStreamer {
	return &FFmpegStreamer{
		hlsDir: hlsDir,
		logger: logger,
	}
}

// StartStream initiates the FFmpeg streaming process.
func (s *FFmpegStreamer) StartStream(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.status {
		s.logger.Warn("Stream already running")
		return nil
	}

	streamPath := filepath.Join(s.hlsDir, "playlist.m3u8")
	s.logger.Infof("Starting stream, outputting to %s", streamPath)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "v4l2", "-i", "/dev/video0",
		"-f", "alsa", "-i", "hw:1,0",
		"-c:v", "h264_omx", // Hardware-accelerated encoder
		"-preset", "veryfast",
		"-maxrate", "2000k",
		"-bufsize", "4000k",
		"-pix_fmt", "yuv420p",
		"-g", "50",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		"-f", "hls",
		"-hls_time", "4",
		"-hls_list_size", "15",
		"-hls_flags", "delete_segments",
		streamPath,
	)

	// Redirect stdout and stderr for logging
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		s.logger.Errorf("Failed to start FFmpeg: %v", err)
		return err
	}

	s.cmd = cmd
	s.status = true
	s.logger.Info("FFmpeg stream started successfully")

	// Monitor the FFmpeg process
	go func() {
		if err := cmd.Wait(); err != nil {
			s.logger.Errorf("FFmpeg process exited with error: %v", err)
		} else {
			s.logger.Info("FFmpeg process exited gracefully")
		}
		s.mutex.Lock()
		s.status = false
		s.mutex.Unlock()
	}()

	return nil
}

// StopStream terminates the FFmpeg streaming process.
func (s *FFmpegStreamer) StopStream() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.status || s.cmd == nil || s.cmd.Process == nil {
		s.logger.Warn("No active stream to stop")
		return nil
	}

	s.logger.Info("Stopping FFmpeg stream...")
	if err := s.cmd.Process.Kill(); err != nil {
		s.logger.Errorf("Failed to kill FFmpeg process: %v", err)
		return err
	}

	s.status = false
	s.logger.Info("FFmpeg stream stopped successfully")
	return nil
}

// IsStreaming returns the current streaming status.
func (s *FFmpegStreamer) IsStreaming() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.status
}