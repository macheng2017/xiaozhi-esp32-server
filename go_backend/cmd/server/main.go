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

	// Setup HTTP server
	http.HandleFunc("/xiaozhi/v1/", internalws.Handler(mqttClient))

	// Create a channel to listen for OS signals (like Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		serverAddr := ":" + cfg.ServerPort
		log.Printf("HTTP server listening on %s", serverAddr)
		if err := http.ListenAndServe(serverAddr, nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", serverAddr, err)
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