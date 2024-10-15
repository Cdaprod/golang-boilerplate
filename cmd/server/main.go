package main

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Cdaprod/multimedia-sys/internal/facade"
	"github.com/Cdaprod/multimedia-sys/internal/gpio"
	"github.com/Cdaprod/multimedia-sys/internal/streaming"
	"github.com/Cdaprod/multimedia-sys/internal/videomanager"
	"github.com/Cdaprod/multimedia-sys/internal/websocket"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

//go:embed web_client/*
var webClientFS embed.FS

// Configuration Constants
const (
	HLSDir          = "/tmp/hls"
	VideoStorageDir = "/mnt/nas/videos"
	GPIOButtonPin   = 18 // BCM pin number
	ServerPort      = ":8080"
)

func main() {
	// Initialize Logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logEntry := logger.WithField("component", "main")

	// Create necessary directories
	if err := os.MkdirAll(HLSDir, 0755); err != nil {
		logEntry.Fatalf("Failed to create HLS directory: %v", err)
	}
	if err := os.MkdirAll(VideoStorageDir, 0755); err != nil {
		logEntry.Fatalf("Failed to create Video Storage directory: %v", err)
	}

	// Initialize Components
	streamer := streaming.NewFFmpegStreamer(HLSDir, logrus.NewEntry(logger))
	wsManager := websocket.NewWebSocketManager(logrus.NewEntry(logger))
	videoManager := videomanager.NewVideoManager(VideoStorageDir, logrus.NewEntry(logger))
	gpioManager := gpio.NewGPIOManager(GPIOButtonPin, 500*time.Millisecond, logrus.NewEntry(logger))

	// Create Facade
	facade := facade.NewFacade(streamer, wsManager, videoManager, gpioManager, logrus.NewEntry(logger))

	// Initialize GPIO
	if err := facade.InitGPIO(); err != nil {
		logEntry.Fatalf("Failed to initialize GPIO: %v", err)
	}
	defer gpioManager.Close()

	// Setup Router
	r := mux.NewRouter()

	// API Endpoints
	r.HandleFunc("/start-stream", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := facade.StartStream(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]string{"status": "Stream started"})
	}).Methods("GET")

	r.HandleFunc("/stop-stream", func(w http.ResponseWriter, r *http.Request) {
		if err := facade.StopStream(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]string{"status": "Stream stopped"})
	}).Methods("GET")

	r.HandleFunc("/list-videos", func(w http.ResponseWriter, r *http.Request) {
		videos, err := facade.ListVideos()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string][]string{"videos": videos})
	}).Methods("GET")

	r.HandleFunc("/videos/{filename}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filename := vars["filename"]
		if err := facade.ServeVideo(filename, w); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}).Methods("GET")

	r.HandleFunc("/ws", facade.RegisterWebSocket).Methods("GET")

	// Serve HLS streams
	r.PathPrefix("/hls/").Handler(http.StripPrefix("/hls/", http.FileServer(http.Dir(HLSDir))))

	// Serve Embedded Web Client
	r.HandleFunc("/", serveWebClient).Methods("GET")
	r.HandleFunc("/{path}", serveWebClient).Methods("GET")

	// Create Server
	srv := &http.Server{
		Handler:      r,
		Addr:         ServerPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Context for GPIO Monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start GPIO Monitoring in a separate goroutine
	go facade.MonitorGPIO(ctx)

	// WaitGroup to handle graceful shutdown
	var wg sync.WaitGroup
	wg.Add(1)

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Start Server in a goroutine
	go func() {
		logEntry.Infof("Server is running on %s", ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logEntry.Fatalf("Server failed: %v", err)
		}
		wg.Done()
	}()

	// Block until a signal is received
	<-stop
	logEntry.Info("Shutting down server...")

	// Create a context with timeout for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logEntry.Fatalf("Server forced to shutdown: %v", err)
	}

	// Cancel GPIO monitoring
	cancel()

	// Wait for server goroutine to finish
	wg.Wait()
	logEntry.Info("Server exited gracefully")
}

// serveWebClient serves the embedded web client files.
func serveWebClient(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	data, err := webClientFS.ReadFile("web_client" + path)
	if err != nil {
		if os.IsNotExist(err) {
			// Serve index.html for SPA routing
			data, err = webClientFS.ReadFile("web_client/index.html")
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(data)
			return
		}
		http.NotFound(w, r)
		return
	}

	// Determine Content-Type
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(path, ".html"):
		contentType = "text/html"
	case strings.HasSuffix(path, ".css"):
		contentType = "text/css"
	case strings.HasSuffix(path, ".js"):
		contentType = "application/javascript"
	case strings.HasSuffix(path, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		contentType = "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		contentType = "image/svg+xml"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

// respondJSON sends a JSON response with appropriate headers.
func respondJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}