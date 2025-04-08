package websocket

import (
	"log"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
	// Assuming you might need config here later, e.g., for allowed origins
	// "github.com/xiaozhi-esp32-server/go_backend/internal/config"
	internalmqtt "github.com/xiaozhi-esp32-server/go_backend/internal/mqtt"
)

// Configure the upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin should be implemented carefully in production
	// to prevent cross-site request forgery attacks.
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections for now
		// TODO: Implement proper origin checking based on configuration
		return true
	},
}

// Handler returns an http.HandlerFunc that handles WebSocket connections
func Handler(mqttClient mqtt.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[WebSocket] Failed to upgrade connection: %v", err)
			return
		}
		defer ws.Close()

		// Log headers for debugging
		log.Printf("[WebSocket] New connection from %s with headers:", ws.RemoteAddr())
		for k, v := range r.Header {
			log.Printf("[WebSocket] Header %s: %v", k, v)
		}

		// Send a welcome message (similar to the "Server Hello" expected by ESP32)
		welcomeMsg := `{"type":"server_hello","status":"ok","transport":"websocket"}`
		if err := ws.WriteMessage(websocket.TextMessage, []byte(welcomeMsg)); err != nil {
			log.Printf("[WebSocket] Failed to send welcome message: %v", err)
		} else {
			log.Printf("[WebSocket] Sent welcome message to %s", ws.RemoteAddr())
		}

		// Track audio frame count for less verbose logging
		audioFrameCount := 0
		totalAudioBytes := 0

		// Read loop
		for {
			messageType, p, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WebSocket] Client %s disconnected unexpectedly: %v", ws.RemoteAddr(), err)
				} else {
					// Don't log "normal" close errors e.g. client closing connection
					if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						log.Printf("[WebSocket] Error reading message from %s: %v", ws.RemoteAddr(), err)
					}
				}
				break // Exit loop on error or disconnect
			}

			// Handle different message types
			switch messageType {
			case websocket.TextMessage:
				// For text messages, log the full content (could be commands or status updates)
				log.Printf("[WebSocket] Received text message from %s: %s", ws.RemoteAddr(), string(p))
				
				// Example: Publish non-binary messages to MQTT topic
				go internalmqtt.Publish(mqttClient, "xiaozhi/ws/ingress", p, 0, false)
			
			case websocket.BinaryMessage:
				// For binary messages (likely audio data), only log count and size
				audioFrameCount++
				totalAudioBytes += len(p)
				
				// Only log every 20 frames to reduce console spam
				if audioFrameCount % 20 == 0 {
					log.Printf("[WebSocket] Received %d audio frames (%d bytes total) from %s", 
						audioFrameCount, totalAudioBytes, ws.RemoteAddr())
				}
				
				// For now, we just echo the binary data back
				// In a real implementation, you might process the audio data or forward it
			}

			// Echo message back (this will later be replaced with actual processing)
			if err := ws.WriteMessage(messageType, p); err != nil {
				log.Printf("[WebSocket] Error writing message to %s: %v", ws.RemoteAddr(), err)
				break // Exit loop on write error
			}
		}

		log.Printf("[WebSocket] Client %s disconnected, received %d audio frames (%d bytes total)", 
			ws.RemoteAddr(), audioFrameCount, totalAudioBytes)
	}
} 