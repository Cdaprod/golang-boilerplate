package videomanager

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// VideoManager defines the interface for video operations.
type VideoManager interface {
	ListVideos() ([]string, error)
	ServeVideo(filename string, w http.ResponseWriter) error
}

// VideoManagerImpl implements the VideoManager interface.
type VideoManagerImpl struct {
	storageDir string
	logger     *logrus.Entry
}

// NewVideoManager creates a new VideoManager instance.
func NewVideoManager(storageDir string, logger *logrus.Entry) *VideoManagerImpl {
	return &VideoManagerImpl{
		storageDir: storageDir,
		logger:     logger,
	}
}

// ListVideos retrieves a list of video filenames from the storage directory.
func (vm *VideoManagerImpl) ListVideos() ([]string, error) {
	var videos []string
	files, err := os.ReadDir(vm.storageDir)
	if err != nil {
		vm.logger.Errorf("Failed to read storage directory: %v", err)
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && isVideoFile(file.Name()) {
			videos = append(videos, file.Name())
		}
	}
	vm.logger.Infof("Found %d videos", len(videos))
	return videos, nil
}

// ServeVideo streams the requested video file to the client.
func (vm *VideoManagerImpl) ServeVideo(filename string, w http.ResponseWriter) error {
	filePath := filepath.Join(vm.storageDir, filename)
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		vm.logger.Warnf("Requested video does not exist: %s", filePath)
		return errors.New("file does not exist")
	}

	http.ServeFile(w, nil, filePath)
	vm.logger.Infof("Serving video: %s", filePath)
	return nil
}

// isVideoFile checks if a file has a video extension.
func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".flv", ".mkv", ".avi":
		return true
	default:
		return false
	}
}