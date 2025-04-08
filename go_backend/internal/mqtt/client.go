package mqtt

import (
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/xiaozhi-esp32-server/go_backend/internal/config"
)

// Client 是 MQTT 客户端的包装器
type Client struct {
	client    paho.Client
	connected bool
}

// NewClient 创建一个新的 MQTT 客户端
func NewClient(cfg *config.Config) *Client {
	log.Printf("[MQTT] Connecting to broker: %s", cfg.MqttBroker)

	opts := paho.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s", cfg.MqttBroker)).
		SetClientID(cfg.MqttClientID).
		SetAutoReconnect(true).
		SetCleanSession(true).
		SetMaxReconnectInterval(10 * time.Second)

	// 如果提供了用户名和密码，则添加到选项中
	if cfg.MqttUsername != "" {
		opts.SetUsername(cfg.MqttUsername)
		if cfg.MqttPassword != "" {
			opts.SetPassword(cfg.MqttPassword)
		}
	}

	// 设置各种回调函数
	opts.SetConnectionLostHandler(func(client paho.Client, err error) {
		log.Printf("[MQTT] Connection lost: %v", err)
	})

	opts.SetOnConnectHandler(func(client paho.Client) {
		log.Printf("[MQTT] Connected to broker")
	})

	client := paho.NewClient(opts)

	mqttClient := &Client{
		client:    client,
		connected: false,
	}

	// 尝试连接，但不阻塞
	go func() {
		for {
			log.Printf("[MQTT] Attempting connection to broker...")
			if token := client.Connect(); token.Wait() && token.Error() != nil {
				log.Printf("[MQTT] Failed to connect to broker: %v. Retrying in background...", token.Error())
				time.Sleep(5 * time.Second)
			} else {
				mqttClient.connected = true
				log.Printf("[MQTT] Successfully connected to broker")
				break
			}
		}
	}()

	return mqttClient
}

// IsConnected 返回客户端是否已连接
func (c *Client) IsConnected() bool {
	return c.connected && c.client.IsConnectionOpen()
}

// Publish 向指定的主题发布消息
func (c *Client) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if !c.IsConnected() {
		log.Printf("[MQTT] WARN: Attempted to publish while not connected.")
		return fmt.Errorf("mqtt client not connected")
	}

	token := c.client.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		log.Printf("[MQTT] Failed to publish to topic %s: %v", topic, token.Error())
		return token.Error()
	}

	return nil
}

// Subscribe 订阅指定的主题
func (c *Client) Subscribe(topic string, qos byte, callback paho.MessageHandler) error {
	if !c.IsConnected() {
		log.Printf("[MQTT] WARN: Attempted to subscribe while not connected.")
		return fmt.Errorf("mqtt client not connected")
	}

	token := c.client.Subscribe(topic, qos, callback)
	if token.Wait() && token.Error() != nil {
		log.Printf("[MQTT] Failed to subscribe to topic %s: %v", topic, token.Error())
		return token.Error()
	}

	log.Printf("[MQTT] Successfully subscribed to topic: %s", topic)
	return nil
}

// Unsubscribe 取消订阅指定的主题
func (c *Client) Unsubscribe(topics ...string) error {
	if !c.IsConnected() {
		log.Printf("[MQTT] WARN: Attempted to unsubscribe while not connected.")
		return fmt.Errorf("mqtt client not connected")
	}

	token := c.client.Unsubscribe(topics...)
	if token.Wait() && token.Error() != nil {
		log.Printf("[MQTT] Failed to unsubscribe from topics %v: %v", topics, token.Error())
		return token.Error()
	}

	log.Printf("[MQTT] Successfully unsubscribed from topics: %v", topics)
	return nil
}

// Disconnect 断开与 MQTT 代理的连接
func (c *Client) Disconnect(quiesce uint) {
	if c.IsConnected() {
		c.client.Disconnect(quiesce)
		c.connected = false
		log.Printf("[MQTT] Disconnected from broker")
	}
} 