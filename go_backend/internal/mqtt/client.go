package mqtt

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/xiaozhi-esp32-server/go_backend/internal/config" // Import config package
)

// Define handlers as package-level variables or within a struct if preferred
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("[MQTT] Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
	// Add logic here to handle incoming MQTT messages.
	// This might involve finding a corresponding WebSocket connection
	// based on the topic or payload and forwarding the message.
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("[MQTT] Connected to Broker")
	// Subscribe to topics upon connection
	subscribeToTopics(client)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("[MQTT] Connection Lost: %v", err)
	// Consider adding reconnection logic here
}

// NewClient initializes and connects the MQTT client
func NewClient(cfg *config.Config) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", cfg.MQTTBroker))
	// Generate a unique client ID
	clientID := "go_backend_server_" + fmt.Sprintf("%d", time.Now().UnixNano())
	opts.SetClientID(clientID)
	// Optional: Set username and password if required by the broker
	// opts.SetUsername("your_username")
	// opts.SetPassword("your_password")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	// Increase PING timeout, helpful for less stable connections
	opts.SetPingTimeout(10 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		// Log fatal might be too harsh, maybe return error?
		log.Printf("[MQTT] Failed to connect to broker: %v. Retrying in background...", token.Error())
		// Don't use log.Fatalf here, allow the server to start anyway
		// The auto-reconnect mechanism will handle retries.
	} else if token.Error() == nil {
		log.Printf("[MQTT] Connection attempt initiated for client ID: %s", clientID)
	} else {
		log.Printf("[MQTT] Connection timed out for client ID: %s. Retrying in background...", clientID)
	}

	return client
}

// subscribeToTopics subscribes the client to relevant MQTT topics
// This should be customized based on your application's needs.
func subscribeToTopics(client mqtt.Client) {
	// Example: Subscribe to a general command topic
	topic := "xiaozhi/commands/all"
	if token := client.Subscribe(topic, 1, nil); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		log.Printf("[MQTT] Failed to subscribe to topic %s: %v", topic, token.Error())
	} else if token.Error() == nil {
		log.Printf("[MQTT] Subscribed to topic: %s", topic)
	}

	// Example: Subscribe to device status updates (using a wildcard)
	statusTopic := "xiaozhi/status/+" // + is a single-level wildcard
	if token := client.Subscribe(statusTopic, 0, messagePubHandler); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		log.Printf("[MQTT] Failed to subscribe to topic %s: %v", statusTopic, token.Error())
	} else if token.Error() == nil {
		log.Printf("[MQTT] Subscribed to topic: %s", statusTopic)
	}

	// Add other necessary subscriptions here
}

// Publish sends a message to a specific MQTT topic
func Publish(client mqtt.Client, topic string, payload interface{}, qos byte, retained bool) error {
	if !client.IsConnected() {
		log.Printf("[MQTT] WARN: Attempted to publish while not connected.")
		// Depending on requirements, you might queue the message or return an error
		return fmt.Errorf("mqtt client not connected")
	}
	token := client.Publish(topic, qos, retained, payload)
	// Optionally wait with timeout for confirmation, especially for QoS > 0
	if success := token.WaitTimeout(2 * time.Second); !success {
		log.Printf("[MQTT] WARN: Timeout waiting for publish confirmation to topic %s", topic)
		// return fmt.Errorf("timeout waiting for publish confirmation")
	}
	if err := token.Error(); err != nil {
		log.Printf("[MQTT] ERROR: Failed to publish to topic %s: %v", topic, err)
		return err
	}
	// log.Printf("[MQTT] Published to topic: %s", topic) // Reduce noise, maybe log only on error/warning
	return nil
} 