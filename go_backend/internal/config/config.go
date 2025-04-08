package config

import (
	"log"
	"os"
)

// Config 结构体包含应用程序的所有配置项
type Config struct {
	// MQTT配置
	MqttBroker   string
	MqttUsername string
	MqttPassword string
	MqttClientID string

	// 服务器配置
	ServerPort string

	// TTS配置
	TTSProvider    string // 默认TTS提供商 (mock, douban)
	DoubanAPIKey   string // 豆包API密钥

	// LLM配置
	LLMProvider    string // 默认LLM提供商 (mock, deepseek)
	DeepseekAPIKey string // Deepseek API密钥
	DeepseekModel  string // Deepseek模型名称
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *Config {
	config := &Config{
		// MQTT 默认值
		MqttBroker:   getEnv("MQTT_BROKER", "localhost:1883"),
		MqttUsername: getEnv("MQTT_USERNAME", ""),
		MqttPassword: getEnv("MQTT_PASSWORD", ""),
		MqttClientID: getEnv("MQTT_CLIENT_ID", "go-backend-server"),

		// 服务器默认值
		ServerPort: getEnv("SERVER_PORT", "8000"),

		// TTS 默认值
		TTSProvider:  getEnv("TTS_PROVIDER", "mock"),
		DoubanAPIKey: getEnv("DOUBAN_API_KEY", ""),

		// LLM 默认值
		LLMProvider:    getEnv("LLM_PROVIDER", "mock"),
		DeepseekAPIKey: getEnv("DEEPSEEK_API_KEY", ""),
		DeepseekModel:  getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
	}

	// 记录配置加载情况
	if os.Getenv("MQTT_BROKER") == "" {
		log.Printf("MQTT_BROKER environment variable not set, using default: %s", config.MqttBroker)
	}
	
	if os.Getenv("SERVER_PORT") == "" {
		log.Printf("SERVER_PORT environment variable not set, using default: %s", config.ServerPort)
	}
	
	if os.Getenv("TTS_PROVIDER") == "" {
		log.Printf("TTS_PROVIDER environment variable not set, using default: %s", config.TTSProvider)
	} else if config.TTSProvider == "douban" && config.DoubanAPIKey == "" {
		log.Printf("Warning: TTS provider set to 'douban' but DOUBAN_API_KEY is not set")
	}
	
	if os.Getenv("LLM_PROVIDER") == "" {
		log.Printf("LLM_PROVIDER environment variable not set, using default: %s", config.LLMProvider)
	} else if config.LLMProvider == "deepseek" && config.DeepseekAPIKey == "" {
		log.Printf("Warning: LLM provider set to 'deepseek' but DEEPSEEK_API_KEY is not set")
	}

	return config
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
} 