package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiaozhi-esp32-server/go_backend/internal/config"
	internalmqtt "github.com/xiaozhi-esp32-server/go_backend/internal/mqtt" // Alias for clarity
	internalws "github.com/xiaozhi-esp32-server/go_backend/internal/websocket"
)

func main() {
	log.Println("Starting Go backend server...")

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize MQTT Client
	mqttClient := internalmqtt.NewClient(cfg)

	// Ensure MQTT client disconnects gracefully on shutdown
	defer func() {
		if mqttClient.IsConnected() {
			log.Println("Disconnecting MQTT client...")
			// Wait 250ms for disconnection process to complete
			mqttClient.Disconnect(250)
			log.Println("MQTT client disconnected.")
		}
	}()

	// Setup WebSocket handler
	wsHandler := internalws.Handler(mqttClient)
	http.HandleFunc("/xiaozhi/v1/", wsHandler)

	// Create a channel to listen for OS signals (like Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP servers in goroutines
	// Primary server on configured port (default: 8000)
	go func() {
		serverAddr := ":" + cfg.ServerPort
		log.Printf("HTTP server listening on %s", serverAddr)
		if err := http.ListenAndServe(serverAddr, nil); err != nil && err != http.ErrServerClosed {
			log.Printf("Could not listen on %s: %v\n", serverAddr, err)
		}
	}()

	// Additional server on port 80 to match ESP32 configuration
	go func() {
		serverAddr := ":80"
		log.Printf("HTTP server also listening on %s (for ESP32 compatibility)", serverAddr)
		if err := http.ListenAndServe(serverAddr, nil); err != nil && err != http.ErrServerClosed {
			log.Printf("Could not listen on %s: %v - This is expected if not running as admin or port 80 is already in use\n", serverAddr, err)
			// We don't exit here, as the main server on port 8000 might still be running
		}
	}()

	// Wait for a termination signal
	<-sigChan
	log.Println("Termination signal received. Shutting down...")

	// Add any other cleanup tasks here (e.g., closing database connections)

	// Brief delay to allow ongoing operations (like MQTT disconnect) to finish
	time.Sleep(500 * time.Millisecond)
	log.Println("Server gracefully stopped.")
} 