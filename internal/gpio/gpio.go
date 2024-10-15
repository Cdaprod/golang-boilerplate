package gpio

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stianeikeland/go-rpio/v4"
)

// GPIOManager defines the interface for GPIO operations.
type GPIOManager interface {
	Init() error
	MonitorButton(ctx context.Context, callback func())
	Close() error
}

// GPIOManagerImpl implements the GPIOManager interface.
type GPIOManagerImpl struct {
	pin      rpio.Pin
	pinNum   int
	logger   *logrus.Entry
	debounce time.Duration
}

// NewGPIOManager creates a new GPIOManager instance.
func NewGPIOManager(pinNum int, debounce time.Duration, logger *logrus.Entry) *GPIOManagerImpl {
	return &GPIOManagerImpl{
		pinNum:   pinNum,
		logger:   logger,
		debounce: debounce,
	}
}

// Init initializes the GPIO pin.
func (g *GPIOManagerImpl) Init() error {
	if err := rpio.Open(); err != nil {
		g.logger.Errorf("Failed to open GPIO: %v", err)
		return err
	}
	g.pin = rpio.Pin(g.pinNum)
	g.pin.Input()
	g.pin.PullUp()
	g.logger.Infof("GPIO pin %d initialized as input with pull-up resistor", g.pinNum)
	return nil
}

// MonitorButton monitors the GPIO pin for button presses with debounce.
func (g *GPIOManagerImpl) MonitorButton(ctx context.Context, callback func()) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var lastState rpio.State
	var lastDebounce time.Time

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("Stopping GPIO button monitoring")
			return
		case <-ticker.C:
			currentState := g.pin.Read()
			if currentState != lastState {
				lastDebounce = time.Now()
			}

			if time.Since(lastDebounce) > g.debounce {
				if currentState != lastState {
					lastState = currentState
					if currentState == rpio.Low {
						g.logger.Info("GPIO button pressed")
						callback()
					}
				}
			}
		}
	}
}

// Close releases the GPIO resources.
func (g *GPIOManagerImpl) Close() error {
	rpio.Close()
	g.logger.Info("GPIO resources closed")
	return nil
}