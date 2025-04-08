package handlers

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/xiaozhi-esp32-server/go_backend/internal/conversation"
	"github.com/xiaozhi-esp32-server/go_backend/internal/llm"
	"github.com/xiaozhi-esp32-server/go_backend/internal/mqtt"
	"github.com/xiaozhi-esp32-server/go_backend/internal/tts"
)

// activeConnections 用于跟踪活跃的会话
var (
	activeConnections = make(map[*websocket.Conn]*conversation.ConversationManager)
	connectionsMutex  sync.Mutex
)

// WebsocketUpgrader 配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有跨域请求（开发环境适用）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler 返回处理 WebSocket 连接的 HTTP 处理函数
func WebSocketHandler(mqttClient *mqtt.Client, llmManager *llm.LLMManager, ttsManager *tts.TTSManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 升级 HTTP 连接为 WebSocket
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[WebSocket] Failed to upgrade connection: %v", err)
			return
		}
		
		// 确保连接最终关闭
		defer func() {
			ws.Close()
			
			// 从活跃会话中移除
			connectionsMutex.Lock()
			if cm, exists := activeConnections[ws]; exists {
				cm.Stop() // 停止会话管理器
				delete(activeConnections, ws)
			}
			connectionsMutex.Unlock()
		}()

		// 记录连接请求头（用于调试）
		log.Printf("[WebSocket] New connection from %s with headers:", ws.RemoteAddr())
		
		// 提取设备ID和客户端ID
		deviceID := ""
		clientID := ""
		
		for k, v := range r.Header {
			log.Printf("[WebSocket] Header %s: %v", k, v)
			
			// 注意：HTTP头是大小写不敏感的，但ESP32可能使用特定大小写
			headerKey := strings.ToLower(k)
			if headerKey == "device-id" && len(v) > 0 {
				deviceID = v[0]
			} else if headerKey == "client-id" && len(v) > 0 {
				clientID = v[0]
			}
		}
		
		log.Printf("[WebSocket] Extracted Device-ID: %s, Client-ID: %s", deviceID, clientID)

		// 创建会话管理器
		cm := conversation.NewConversationManager(ws, llmManager, ttsManager, deviceID, clientID)
		
		// 保存到活跃连接中
		connectionsMutex.Lock()
		activeConnections[ws] = cm
		connectionsMutex.Unlock()
		
		// 启动会话（会发送欢迎消息并开始沉默检测）
		cm.Start()

		// 消息接收循环
		for {
			messageType, p, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WebSocket] Client %s disconnected unexpectedly: %v", ws.RemoteAddr(), err)
				} else if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					// 只记录非正常关闭错误
					log.Printf("[WebSocket] Error reading message from %s: %v", ws.RemoteAddr(), err)
				}
				break // 出错就跳出循环，连接将在 defer 中关闭
			}

			// 根据消息类型处理
			switch messageType {
			case websocket.TextMessage:
				// 处理文本消息（通常是控制命令或状态更新）
				err = cm.HandleTextMessage(p)
				if err != nil {
					log.Printf("[WebSocket] Error handling text message: %v", err)
				}
				
				// 发布到 MQTT（如果需要且连接有效）
				if mqttClient != nil && mqttClient.IsConnected() {
					err := mqttClient.Publish("xiaozhi/ws/ingress", 0, false, p)
					if err != nil {
						log.Printf("[WebSocket] Error publishing to MQTT: %v", err)
					}
				}
				
			case websocket.BinaryMessage:
				// 处理二进制消息（通常是音频数据）
				err = cm.HandleBinaryMessage(p)
				if err != nil {
					log.Printf("[WebSocket] Error handling binary message: %v", err)
				}
			}
		}

		// 连接结束，输出统计信息
		log.Printf("[WebSocket] Connection from %s closed", ws.RemoteAddr())
	}
}

// GetActiveConnectionsCount 返回当前活跃的 WebSocket 连接数
func GetActiveConnectionsCount() int {
	connectionsMutex.Lock()
	defer connectionsMutex.Unlock()
	return len(activeConnections)
}

// BroadcastTextMessage 向所有活跃连接广播文本消息
func BroadcastTextMessage(message []byte) {
	connectionsMutex.Lock()
	defer connectionsMutex.Unlock()
	
	for conn := range activeConnections {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("[WebSocket] Error broadcasting to %s: %v", conn.RemoteAddr(), err)
		}
	}
} 