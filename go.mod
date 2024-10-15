module github.com/Cdaprod/multimedia-sys

go 1.20

require (
    github.com/gorilla/mux v1.8.0    // For routing
    github.com/gorilla/websocket v1.5.0  // For WebSocket handling
    github.com/sirupsen/logrus v1.8.1    // For structured logging
    github.com/stianeikeland/go-rpio/v4 v4.6.0   // For GPIO handling on Raspberry Pi
)

replace github.com/stianeikeland/go-rpio/v4 => github.com/stianeikeland/go-rpio v4.6.0 // Required to ensure compatibility with v4