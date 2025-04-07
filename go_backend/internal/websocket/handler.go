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

		// Extract client identifier if possible (e.g., from URL path or query param)
		// clientID := r.URL.Query().Get("clientId")
		// if clientID == "" {
		// 	log.Printf("[WebSocket] Client connected without ID from %s", ws.RemoteAddr())
		// 	// Optionally close connection if ID is required
		// 	// ws.Close()
		// 	// return
		// } else {
		// 	log.Printf("[WebSocket] Client '%s' connected from %s", clientID, ws.RemoteAddr())
		// }
		log.Printf("[WebSocket] Client connected from %s", ws.RemoteAddr())

		// TODO: Register the connection (e.g., in a map[string]*websocket.Conn)
		// associated with its clientID for targeted MQTT message forwarding.

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

			log.Printf("[WebSocket] Received message (type %d) from %s: %s", messageType, ws.RemoteAddr(), p)

			// --- Message Processing Logic --- 
			// 1. Determine message type (e.g., JSON command, binary audio)
			// 2. Parse the message
			// 3. Decide action: Echo, Publish to MQTT, process internally, etc.

			// Example: Publish non-binary messages to a default MQTT topic
			if messageType == websocket.TextMessage {
				// Modify topic based on clientID or message content if needed
				go internalmqtt.Publish(mqttClient, "xiaozhi/ws/ingress", p, 0, false)
			}

			// Echo message back (remove or modify based on requirements)
			if err := ws.WriteMessage(messageType, p); err != nil {
				log.Printf("[WebSocket] Error writing message to %s: %v", ws.RemoteAddr(), err)
				break // Exit loop on write error
			}
		}

		log.Printf("[WebSocket] Client %s disconnected", ws.RemoteAddr())
		// TODO: Unregister the connection
	}
} 