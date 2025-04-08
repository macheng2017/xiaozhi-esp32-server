package conversation

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xiaozhi-esp32-server/go_backend/internal/llm"
	"github.com/xiaozhi-esp32-server/go_backend/internal/models"
	"github.com/xiaozhi-esp32-server/go_backend/internal/tts"
	"github.com/google/uuid"
)

// ConversationManager 管理与单个客户端的会话状态
type ConversationManager struct {
	// WebSocket 连接
	conn *websocket.Conn
	
	// 会话状态
	currentState models.ListenState
	
	// 音频数据统计
	audioFrameCount int
	totalAudioBytes int
	lastAudioTime   time.Time
	
	// 配置
	silenceDuration time.Duration // 检测用户停止说话的沉默时长
	minAudioFrames  int           // 处理语音所需的最小帧数
	
	// 同步
	mu              sync.Mutex
	stopDetection   chan struct{}
	
	// 客户端信息
	clientID        string 
	deviceID        string
	clientIP        string
	
	// LLM 和 TTS 服务
	llmManager      *llm.LLMManager
	ttsManager      *tts.TTSManager
	
	// 聊天历史
	chatHistory     []llm.Message
}

// NewConversationManager 创建一个新的会话管理器
func NewConversationManager(conn *websocket.Conn, llmManager *llm.LLMManager, ttsManager *tts.TTSManager, deviceID, clientID string) *ConversationManager {
	// 创建初始系统提示
	systemPrompt := "你是一个友好的语音助手，请简短、清晰地回答用户问题。回应应直接、有帮助，避免不必要的冗长解释。"
	
	return &ConversationManager{
		conn:            conn,
		currentState:    models.StateIdle,
		audioFrameCount: 0,
		totalAudioBytes: 0,
		lastAudioTime:   time.Now(),
		silenceDuration: 1500 * time.Millisecond, // 1.5 秒的沉默判定为用户停止说话
		minAudioFrames:  20,                     // 至少需要 20 帧音频才处理
		stopDetection:   make(chan struct{}),
		clientIP:        conn.RemoteAddr().String(),
		deviceID:        deviceID,
		clientID:        clientID,
		llmManager:      llmManager,
		ttsManager:      ttsManager,
		chatHistory: []llm.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
		},
	}
}

// Start 启动会话管理
func (cm *ConversationManager) Start() {
	// 发送服务器欢迎消息
	cm.sendServerHello()
	
	// 启动沉默检测
	go cm.runSilenceDetection()
}

// Stop 停止会话管理并清理资源
func (cm *ConversationManager) Stop() {
	close(cm.stopDetection)
	log.Printf("[Conversation] Stopped conversation manager for client %s", cm.clientIP)
}

// HandleBinaryMessage 处理二进制消息（通常是音频数据）
func (cm *ConversationManager) HandleBinaryMessage(data []byte) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 更新音频统计
	cm.audioFrameCount++
	cm.totalAudioBytes += len(data)
	cm.lastAudioTime = time.Now()
	
	// 如果是空闲状态，转为监听状态
	if cm.currentState == models.StateIdle {
		cm.currentState = models.StateListening
		log.Printf("[Conversation] State changed to LISTENING for client %s", cm.clientIP)
		
		// 发送监听开始消息
		err := cm.sendListeningStartMessage()
		if err != nil {
			log.Printf("[Conversation] Error sending listening start message: %v", err)
		}
	}
	
	// 只在接收到一定数量的帧时记录日志，减少刷屏
	if cm.audioFrameCount%20 == 0 {
		log.Printf("[Conversation] Received %d audio frames (%d bytes) from %s", 
			cm.audioFrameCount, cm.totalAudioBytes, cm.clientIP)
	}
	
	// 此处应将音频数据发送到语音识别服务
	// 当前仅回显音频数据（在实际应用中应移除）
	return cm.conn.WriteMessage(websocket.BinaryMessage, data)
}

// HandleTextMessage 处理文本消息（通常是控制命令或状态更新）
func (cm *ConversationManager) HandleTextMessage(data []byte) error {
	// 先记录原始消息
	log.Printf("[Conversation] Received text message from %s: %s", cm.clientIP, string(data))
	
	// 尝试解析 JSON
	var jsonMsg map[string]interface{}
	if err := json.Unmarshal(data, &jsonMsg); err != nil {
		log.Printf("[Conversation] Failed to parse JSON: %v", err)
		return nil // 非 JSON 消息，忽略
	}
	
	// 提取消息类型
	msgTypeStr, ok := jsonMsg["type"].(string)
	if !ok {
		return nil // 没有类型字段，忽略
	}
	
	msgType := models.MessageType(msgTypeStr)
	log.Printf("[Conversation] Processed message type: %s", msgType)
	
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// 根据消息类型处理
	switch msgType {
	case models.TypeHello:
		// 客户端 hello 消息，可能包含能力信息
		cm.processClientHello(jsonMsg)
		
	case models.TypeListeningStart:
		// 用户开始说话
		log.Printf("[Conversation] User started speaking (explicit notification)")
		cm.audioFrameCount = 0
		cm.totalAudioBytes = 0
		cm.currentState = models.StateListening
		
	case models.TypeListeningStop:
		// 用户明确停止说话
		log.Printf("[Conversation] User stopped speaking (explicit notification)")
		if cm.audioFrameCount >= cm.minAudioFrames {
			cm.processUserSpeech()
		} else {
			log.Printf("[Conversation] Not enough audio (%d frames), ignoring", cm.audioFrameCount)
			cm.currentState = models.StateIdle
		}
		
	case models.TypeText:
		// 处理文本消息（模拟语音识别结果）
		textContent, ok := jsonMsg["text"].(string)
		if ok && textContent != "" {
			log.Printf("[Conversation] Processing text message: %s", textContent)
			
			// 临时将状态设为思考中
			cm.currentState = models.StateThinking
			cm.sendThinkingResponse()
			
			// 处理用户文本消息
			go cm.processUserText(textContent)
		}
	}
	
	return nil
}

// GetState 返回当前会话状态
func (cm *ConversationManager) GetState() models.ListenState {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.currentState
}

// GetClientInfo 返回客户端信息
func (cm *ConversationManager) GetClientInfo() (clientID, deviceID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.clientID, cm.deviceID
}

// 内部方法

// runSilenceDetection 运行沉默检测
func (cm *ConversationManager) runSilenceDetection() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.mu.Lock()
			if cm.currentState == models.StateListening &&
				time.Since(cm.lastAudioTime) > cm.silenceDuration &&
				cm.audioFrameCount >= cm.minAudioFrames {
				
				log.Printf("[Conversation] Silence detected after %d frames for %s", 
					cm.audioFrameCount, cm.clientIP)
				
				// 处理用户的语音
				cm.processUserSpeech()
			}
			cm.mu.Unlock()
			
		case <-cm.stopDetection:
			return
		}
	}
}

// processUserSpeech 处理用户语音
func (cm *ConversationManager) processUserSpeech() {
	// 发送思考中状态
	cm.sendThinkingResponse()
	cm.currentState = models.StateThinking
	
	// 模拟语音识别过程
	recognizedText := "这是模拟的语音识别结果，实际应用中应对音频进行真实的语音识别。"
	
	go cm.processUserText(recognizedText)
}

// processUserText 处理用户文本（来自语音识别或直接文本输入）
func (cm *ConversationManager) processUserText(text string) {
	// 将用户消息添加到聊天历史
	userMessage := llm.Message{
		Role:    "user",
		Content: text,
	}
	
	cm.mu.Lock()
	cm.chatHistory = append(cm.chatHistory, userMessage)
	history := append([]llm.Message{}, cm.chatHistory...) // 复制一份历史记录
	cm.mu.Unlock()
	
	var llmResponse string
	
	// 调用LLM获取响应
	if cm.llmManager != nil {
		// 使用流式响应方式获取大模型回复
		err := cm.llmManager.StreamChat(history, nil, func(chunk *llm.ResponseChunk) error {
			// 可以实现流式输出，这里简化处理
			if chunk.Content != "" {
				log.Printf("[Conversation] LLM chunk: %s", chunk.Content)
			}
			return nil
		})
		
		if err != nil {
			log.Printf("[Conversation] Error getting LLM response: %v", err)
			llmResponse = "抱歉，我暂时无法回答您的问题。请稍后再试。"
		} else {
			// 获取完整响应（通过非流式方法）
			response, err := cm.llmManager.Chat(history, nil)
			if err != nil {
				log.Printf("[Conversation] Error getting LLM response: %v", err)
				llmResponse = "抱歉，我暂时无法回答您的问题。请稍后再试。"
			} else {
				llmResponse = response.Content
			}
		}
	} else {
		// 如果LLM管理器不可用，生成随机响应
		llmResponse = cm.generateRandomResponse()
	}
	
	// 保存助手回复到历史记录
	assistantMessage := llm.Message{
		Role:    "assistant",
		Content: llmResponse,
	}
	
	cm.mu.Lock()
	cm.chatHistory = append(cm.chatHistory, assistantMessage)
	
	// 限制历史记录长度（保留系统消息和最近的10组对话）
	if len(cm.chatHistory) > 21 { // 1个系统消息 + 10组用户和助手消息
		// 保留第一条系统消息
		cm.chatHistory = append(cm.chatHistory[:1], cm.chatHistory[len(cm.chatHistory)-20:]...)
	}
	cm.mu.Unlock()
	
	// 发送响应文本前的状态变更
	cm.mu.Lock()
	cm.sendSpeakingResponse()
	cm.currentState = models.StateSpeaking
	cm.mu.Unlock()
	
	// 发送文本响应消息
	cm.sendTextResponse(llmResponse)
	
	// 如果TTS管理器可用，合成语音
	var audioData []byte
	var ttsErr error
	
	if cm.ttsManager != nil {
		audioData, ttsErr = cm.ttsManager.SynthesizeSpeech(llmResponse, map[string]string{
			"voice_id": "zh_female_qingxin", // 默认声音
			"format":   "mp3",
		})
		
		if ttsErr != nil {
			log.Printf("[Conversation] Error synthesizing speech: %v", ttsErr)
		}
	}
	
	// 如果合成成功，发送音频数据
	if audioData != nil && len(audioData) > 0 {
		// 分块发送大音频文件，每块最大32KB
		chunkSize := 32 * 1024 // 32KB
		
		for i := 0; i < len(audioData); i += chunkSize {
			end := i + chunkSize
			if end > len(audioData) {
				end = len(audioData)
			}
			
			cm.mu.Lock()
			err := cm.conn.WriteMessage(websocket.BinaryMessage, audioData[i:end])
			cm.mu.Unlock()
			
			if err != nil {
				log.Printf("[Conversation] Error sending audio chunk: %v", err)
				break
			}
			
			// 短暂暂停，避免发送过快
			time.Sleep(50 * time.Millisecond)
		}
	} else {
		// 模拟TTS延迟
		time.Sleep(1 * time.Second)
	}
	
	// 恢复到空闲状态
	cm.mu.Lock()
	cm.sendIdleResponse()
	cm.currentState = models.StateIdle
	cm.audioFrameCount = 0
	cm.mu.Unlock()
}

// processClientHello 处理客户端的 hello 消息
func (cm *ConversationManager) processClientHello(data map[string]interface{}) {
	// 提取客户端和设备 ID
	if transportData, ok := data["transport"].(string); ok {
		log.Printf("[Conversation] Client transport: %s", transportData)
	}
	
	// 如果deviceID为空，从消息中提取
	if cm.deviceID == "" {
		if deviceID, ok := data["device_id"].(string); ok {
			cm.deviceID = deviceID
			log.Printf("[Conversation] Device ID from message: %s", deviceID)
		}
	} else {
		log.Printf("[Conversation] Using Device ID from header: %s", cm.deviceID)
	}
	
	// 如果clientID为空，从消息中提取
	if cm.clientID == "" {
		if clientID, ok := data["client_id"].(string); ok {
			cm.clientID = clientID
			log.Printf("[Conversation] Client ID from message: %s", clientID)
		}
	} else {
		log.Printf("[Conversation] Using Client ID from header: %s", cm.clientID)
	}
	
	// 处理音频参数
	if audioParams, ok := data["audio_params"].(map[string]interface{}); ok {
		log.Printf("[Conversation] Received audio params: %v", audioParams)
	}
	
	log.Printf("[Conversation] Processed client hello message")
}

// sendServerHello 发送服务器欢迎消息
func (cm *ConversationManager) sendServerHello() {
	sessionID := uuid.New().String()
	
	msg := models.ServerHelloMessage{
		Type:      models.TypeServerHello,
		Status:    "ok",
		Transport: "websocket",
		SessionID: sessionID,
	}
	
	jsonData, _ := json.Marshal(msg)
	
	if err := cm.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Printf("[Conversation] Failed to send welcome: %v", err)
	} else {
		log.Printf("[Conversation] Sent welcome to %s with session ID: %s", cm.clientIP, sessionID)
	}
}

// sendThinkingResponse 发送思考中状态
func (cm *ConversationManager) sendThinkingResponse() {
	resp := models.ListenMessage{
		Type:  models.TypeListen,
		State: models.StateThinking,
	}
	
	jsonData, _ := json.Marshal(resp)
	log.Printf("[Conversation] Sending thinking state to %s", cm.clientIP)
	
	if err := cm.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Printf("[Conversation] Error sending thinking state: %v", err)
	}
}

// sendSpeakingResponse 发送说话中状态
func (cm *ConversationManager) sendSpeakingResponse() {
	resp := models.ListenMessage{
		Type:  models.TypeListen,
		State: models.StateSpeaking,
	}
	
	jsonData, _ := json.Marshal(resp)
	log.Printf("[Conversation] Sending speaking state to %s", cm.clientIP)
	
	if err := cm.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Printf("[Conversation] Error sending speaking state: %v", err)
	}
}

// sendIdleResponse 发送空闲状态
func (cm *ConversationManager) sendIdleResponse() {
	resp := models.ListenMessage{
		Type:  models.TypeListen,
		State: models.StateIdle,
	}
	
	jsonData, _ := json.Marshal(resp)
	log.Printf("[Conversation] Sending idle state to %s", cm.clientIP)
	
	if err := cm.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Printf("[Conversation] Error sending idle state: %v", err)
	}
}

// sendListeningStartMessage 发送监听开始消息
func (cm *ConversationManager) sendListeningStartMessage() error {
	msg := models.SimpleMessage{
		Type: models.TypeListeningStart,
	}
	
	jsonData, _ := json.Marshal(msg)
	
	return cm.conn.WriteMessage(websocket.TextMessage, jsonData)
}

// sendTextResponse 发送文本响应
func (cm *ConversationManager) sendTextResponse(text string) {
	msg := struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		Type: "text",
		Text: text,
	}
	
	jsonData, _ := json.Marshal(msg)
	log.Printf("[Conversation] Sending text response to %s", cm.clientIP)
	
	if err := cm.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Printf("[Conversation] Error sending text response: %v", err)
	}
}

// generateRandomResponse 生成随机响应（用于测试）
func (cm *ConversationManager) generateRandomResponse() string {
	responses := []string{
		"你好！我是模拟的语音助手，正在测试WebSocket连接。",
		"我收到了你的语音消息，但我还没有真正的语音识别功能。",
		"音频数据已成功传输，WebSocket通信正常工作。",
		"这是一个测试回复，实际部署时会被真实的AI助手响应替代。",
		"我是一个简单的模拟聊天机器人，用于测试语音交互流程。",
	}
	
	return responses[rand.Intn(len(responses))]
} 