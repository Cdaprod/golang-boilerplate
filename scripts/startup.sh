#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to handle termination signals
term_handler() {
    echo "Termination signal received. Shutting down..."
    # Stop Nginx
    nginx -s quit
    # Stop the Go application
    kill -TERM "$GO_PID"
    wait "$GO_PID"
    exit 0
}

# Trap termination signals
trap 'term_handler' SIGTERM SIGINT

# Start Nginx in the background
echo "Starting Nginx..."
/usr/local/nginx/sbin/nginx

# Start the multimedia server (cmd/server/main.go)
echo "Starting multimedia server..."
/usr/local/bin/multimedia-sys &
GO_PID=$!

# Wait for the Go application to exit
wait "$GO_PID"