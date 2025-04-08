package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiaozhi-esp32-server/go_backend/internal/config"
	"github.com/xiaozhi-esp32-server/go_backend/internal/handlers"
	"github.com/xiaozhi-esp32-server/go_backend/internal/llm"
	internalmqtt "github.com/xiaozhi-esp32-server/go_backend/internal/mqtt"
	"github.com/xiaozhi-esp32-server/go_backend/internal/tts"
)

func main() {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())
	
	log.Println("Starting Go backend server...")
	startTime := time.Now()

	// 加载配置
	cfg := config.LoadConfig()

	// 初始化 MQTT 客户端
	mqttClient := internalmqtt.NewClient(cfg)

	// 初始化 TTS 管理器
	ttsManager := tts.NewTTSManager()
	
	// 注册TTS提供商
	if cfg.TTSProvider == "douban" && cfg.DoubanAPIKey != "" {
		doubanProvider := tts.NewDoubanTTSProvider(cfg.DoubanAPIKey)
		ttsManager.RegisterProvider("douban", doubanProvider)
		ttsManager.SetDefaultProvider("douban")
	} else {
		// 默认使用模拟TTS提供商
		mockProvider := tts.NewMockProvider()
		ttsManager.RegisterProvider("mock", mockProvider)
	}
	
	// 初始化TTS管理器
	if err := ttsManager.Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize TTS: %v", err)
	} else {
		log.Printf("TTS manager initialized with provider: %s", cfg.TTSProvider)
	}
	
	// 初始化 LLM 管理器
	llmManager := llm.NewLLMManager()
	
	// 注册LLM提供商
	if cfg.LLMProvider == "deepseek" && cfg.DeepseekAPIKey != "" {
		deepseekProvider := llm.NewDeepseekProvider(cfg.DeepseekAPIKey, cfg.DeepseekModel)
		llmManager.RegisterProvider("deepseek", deepseekProvider)
		llmManager.SetDefaultProvider("deepseek")
	} else {
		// 默认使用模拟LLM提供商
		mockProvider := llm.NewMockProvider("模拟大语言模型")
		llmManager.RegisterProvider("mock", mockProvider)
	}
	
	// 初始化LLM管理器
	if err := llmManager.Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize LLM: %v", err)
	} else {
		log.Printf("LLM manager initialized with provider: %s", cfg.LLMProvider)
	}

	// 确保资源正常清理
	defer func() {
		// 关闭 MQTT 客户端连接
		if mqttClient.IsConnected() {
			log.Println("正在断开 MQTT 客户端连接...")
			mqttClient.Disconnect(250) // 等待 250ms 完成断开流程
			log.Println("MQTT 客户端已断开连接。")
		}
	}()

	// 设置 HTTP 路由
	// 主要的 WebSocket 路由
	http.HandleFunc("/xiaozhi/v1/", handlers.WebSocketHandler(mqttClient, llmManager, ttsManager))
	
	// 健康检查端点
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// 状态信息端点
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// 服务状态信息
		statusInfo := struct {
			Status            string    `json:"status"`
			ActiveConnections int       `json:"active_connections"`
			MqttConnected     bool      `json:"mqtt_connected"`
			ServerStartTime   time.Time `json:"server_start_time"`
			Uptime            string    `json:"uptime"`
			Version           string    `json:"version"`
			TTSProvider       string    `json:"tts_provider"`
			LLMProvider       string    `json:"llm_provider"`
		}{
			Status:            "running",
			ActiveConnections: handlers.GetActiveConnectionsCount(),
			MqttConnected:     mqttClient.IsConnected(),
			ServerStartTime:   startTime,
			Uptime:            time.Since(startTime).String(),
			Version:           "1.0.0", 
			TTSProvider:       cfg.TTSProvider,
			LLMProvider:       cfg.LLMProvider,
		}
		
		// 将状态信息编码为 JSON 并写入响应
		if err := json.NewEncoder(w).Encode(statusInfo); err != nil {
			log.Printf("Error encoding status info: %v", err)
		}
	})

	// TTS API 端点
	http.HandleFunc("/api/tts/voices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// 获取当前默认TTS提供商的声音列表
		provider, err := ttsManager.GetProvider(cfg.TTSProvider)
		if err != nil {
			http.Error(w, "Failed to get TTS provider: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		voices, err := provider.GetVoices()
		if err != nil {
			http.Error(w, "Failed to get voices: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(voices)
	})

	// 创建用于监听操作系统信号的通道（如 Ctrl+C）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在协程中启动 HTTP 服务器
	// 主服务器使用配置的端口（默认：8000）
	go func() {
		serverAddr := ":" + cfg.ServerPort
		log.Printf("HTTP 服务器在 %s 端口启动", serverAddr)
		if err := http.ListenAndServe(serverAddr, nil); err != nil && err != http.ErrServerClosed {
			log.Printf("无法在 %s 端口上监听: %v\n", serverAddr, err)
		}
	}()

	// 尝试在端口 80 上启动额外的服务器（为了兼容性，需要管理员权限）
	go func() {
		serverAddr := ":80"
		log.Printf("HTTP 服务器也尝试在 %s 端口启动（为了 ESP32 兼容性）", serverAddr)
		if err := http.ListenAndServe(serverAddr, nil); err != nil && err != http.ErrServerClosed {
			log.Printf("无法在 %s 端口上监听: %v - 这是意料之中的，如果没有管理员权限或端口 80 已被占用\n", serverAddr, err)
			// 我们不在这里退出，因为主服务器在端口 8000 上可能仍在运行
		}
	}()

	// 等待终止信号
	<-sigChan
	log.Println("收到终止信号。正在关闭...")

	// 在这里添加其他清理任务（例如，关闭数据库连接）

	// 短暂延迟，允许正在进行的操作（如 MQTT 断开连接）完成
	time.Sleep(500 * time.Millisecond)
	log.Println("服务器已优雅停止。")
} 