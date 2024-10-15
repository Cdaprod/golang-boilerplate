package facade

import (
	"context"

	"github.com/Cdaprod/multimedia-sys/internal/gpio"
	"github.com/Cdaprod/multimedia-sys/internal/streaming"
	"github.com/Cdaprod/multimedia-sys/internal/videomanager"
	"github.com/Cdaprod/multimedia-sys/internal/websocket"
	"github.com/sirupsen/logrus"
)

// Facade defines the interface for interacting with all subsystems.
type Facade interface {
	StartStream(ctx context.Context) error
	StopStream() error
	IsStreaming() bool
	ListVideos() ([]string, error)
	ServeVideo(filename string, w http.ResponseWriter) error
	BroadcastMessage(message string)
	RegisterWebSocket(w http.ResponseWriter, r *http.Request)
	InitGPIO() error
	MonitorGPIO(ctx context.Context)
}

// facadeImpl implements the Facade interface.
type facadeImpl struct {
	streamer    streaming.Streamer
	wsManager   websocket.WebSocketManager
	videoManager videomanager.VideoManager
	gpioManager gpio.GPIOManager
	logger      *logrus.Entry
}

// NewFacade creates a new Facade instance.
func NewFacade(streamer streaming.Streamer, wsManager websocket.WebSocketManager, videoManager videomanager.VideoManager, gpioManager gpio.GPIOManager, logger *logrus.Entry) Facade {
	return &facadeImpl{
		streamer:    streamer,
		wsManager:   wsManager,
		videoManager: videoManager,
		gpioManager: gpioManager,
		logger:      logger,
	}
}

// StartStream initiates the streaming process.
func (f *facadeImpl) StartStream(ctx context.Context) error {
	f.logger.Info("Facade: Starting stream")
	err := f.streamer.StartStream(ctx)
	if err != nil {
		f.logger.Errorf("Facade: Failed to start stream: %v", err)
		return err
	}
	f.BroadcastMessage("Stream started")
	return nil
}

// StopStream terminates the streaming process.
func (f *facadeImpl) StopStream() error {
	f.logger.Info("Facade: Stopping stream")
	err := f.streamer.StopStream()
	if err != nil {
		f.logger.Errorf("Facade: Failed to stop stream: %v", err)
		return err
	}
	f.BroadcastMessage("Stream stopped")
	return nil
}

// IsStreaming checks if streaming is active.
func (f *facadeImpl) IsStreaming() bool {
	return f.streamer.IsStreaming()
}

// ListVideos retrieves the list of available videos.
func (f *facadeImpl) ListVideos() ([]string, error) {
	f.logger.Info("Facade: Listing videos")
	return f.videoManager.ListVideos()
}

// ServeVideo streams the specified video to the client.
func (f *facadeImpl) ServeVideo(filename string, w http.ResponseWriter) error {
	f.logger.Infof("Facade: Serving video %s", filename)
	return f.videoManager.ServeVideo(filename, w)
}

// BroadcastMessage sends a message to all connected WebSocket clients.
func (f *facadeImpl) BroadcastMessage(message string) {
	f.logger.Infof("Facade: Broadcasting message: %s", message)
	f.wsManager.BroadcastMessage(message)
}

// RegisterWebSocket handles WebSocket connection upgrades and management.
func (f *facadeImpl) RegisterWebSocket(w http.ResponseWriter, r *http.Request) {
	f.wsManager.HandleWebSocket(w, r)
}

// InitGPIO initializes the GPIO manager.
func (f *facadeImpl) InitGPIO() error {
	f.logger.Info("Facade: Initializing GPIO")
	return f.gpioManager.Init()
}

// MonitorGPIO starts monitoring the GPIO button for interactions.
func (f *facadeImpl) MonitorGPIO(ctx context.Context) {
	f.logger.Info("Facade: Starting GPIO monitoring")
	f.gpioManager.MonitorButton(ctx, func() {
		if f.IsStreaming() {
			if err := f.StopStream(); err != nil {
				f.logger.Errorf("Facade: Error stopping stream via GPIO: %v", err)
			}
		} else {
			if err := f.StartStream(ctx); err != nil {
				f.logger.Errorf("Facade: Error starting stream via GPIO: %v", err)
			}
		}
	})
}