package config

import (
	"log"
	"os"
)

// Config holds the application configuration
type Config struct {
	MQTTBroker string
	ServerPort string
}

// LoadConfig loads configuration from environment variables or defaults
func LoadConfig() *Config {
	mqttBroker := os.Getenv("MQTT_BROKER")
	if mqttBroker == "" {
		mqttBroker = "localhost:1883" // Default MQTT broker
		log.Printf("MQTT_BROKER environment variable not set, using default: %s", mqttBroker)
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8000" // Default server port
		log.Printf("SERVER_PORT environment variable not set, using default: %s", serverPort)
	}

	return &Config{
		MQTTBroker: mqttBroker,
		ServerPort: serverPort,
	}
} 