package llm

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// MockProvider 是一个简单的模拟 LLM 提供商，用于开发和测试
type MockProvider struct {
	initialized bool
	name        string
}

// NewMockProvider 创建一个新的模拟 LLM 提供商
func NewMockProvider(name string) *MockProvider {
	if name == "" {
		name = "默认模拟模型"
	}
	return &MockProvider{
		name: name,
	}
}

// Chat 模拟非流式对话处理
func (p *MockProvider) Chat(messages []Message, options map[string]interface{}) (*Response, error) {
	if !p.initialized {
		return nil, ErrNotInitialized
	}
	
	// 记录收到的消息
	log.Printf("[LLM:Mock] Received Chat request with %d messages", len(messages))
	for i, msg := range messages {
		log.Printf("[LLM:Mock] Message %d - Role: %s, Content: %s", i, msg.Role, truncateString(msg.Content, 50))
	}
	
	// 选择最后一条用户消息
	userMessage := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userMessage = messages[i].Content
			break
		}
	}
	
	// 生成模拟响应
	responseContent := generateMockResponse(userMessage)
	
	// 返回响应
	return &Response{
		Content:     responseContent,
		FinishReason: "stop",
		Metadata: map[string]interface{}{
			"model":        p.name,
			"response_time": time.Now().Unix(),
		},
	}, nil
}

// StreamChat 模拟流式对话处理
func (p *MockProvider) StreamChat(messages []Message, options map[string]interface{}, callback StreamCallback) error {
	if !p.initialized {
		return ErrNotInitialized
	}
	
	// 记录收到的消息
	log.Printf("[LLM:Mock] Received StreamChat request with %d messages", len(messages))
	
	// 选择最后一条用户消息
	userMessage := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userMessage = messages[i].Content
			break
		}
	}
	
	// 生成模拟响应
	responseContent := generateMockResponse(userMessage)
	
	// 将响应分成多个块，模拟流式传输
	chunks := splitIntoChunks(responseContent, 10) // 每块约10个字符
	
	for i, chunk := range chunks {
		isFinal := i == len(chunks)-1
		
		// 创建响应块
		responseChunk := &ResponseChunk{
			Content: chunk,
			IsFinal: isFinal,
		}
		
		if isFinal {
			responseChunk.FinishReason = "stop"
		}
		
		// 调用回调函数处理块
		if err := callback(responseChunk); err != nil {
			return fmt.Errorf("error in callback: %w", err)
		}
		
		// 添加一点延迟，模拟网络延迟
		time.Sleep(100 * time.Millisecond)
	}
	
	return nil
}

// Initialize 初始化模拟 LLM 提供商
func (p *MockProvider) Initialize() error {
	log.Printf("[LLM:Mock] Initializing mock LLM provider: %s", p.name)
	p.initialized = true
	return nil
}

// Cleanup 清理模拟 LLM 提供商资源
func (p *MockProvider) Cleanup() error {
	log.Printf("[LLM:Mock] Cleaning up mock LLM provider")
	p.initialized = false
	return nil
}

// 辅助函数

// generateMockResponse 根据用户消息生成模拟响应
func generateMockResponse(userMessage string) string {
	// 如果消息中包含问候，返回问候
	userMessage = strings.ToLower(userMessage)
	if strings.Contains(userMessage, "你好") || strings.Contains(userMessage, "hello") || strings.Contains(userMessage, "hi") {
		return "你好！我是一个模拟的语音助手模型。我可以帮助你回答问题、提供信息或者与你聊天。有什么我可以帮助你的吗？"
	}
	
	// 如果消息中包含询问身份
	if strings.Contains(userMessage, "你是谁") || strings.Contains(userMessage, "你的名字") {
		return "我是一个模拟的语音助手模型，用于测试语音交互系统。我不是真正的AI助手，只是为了开发测试而创建的简单模拟程序。"
	}
	
	// 如果消息中包含询问时间
	if strings.Contains(userMessage, "时间") || strings.Contains(userMessage, "几点") {
		return fmt.Sprintf("现在的时间是 %s。请注意，这是一个模拟响应，实际时间可能不准确。", time.Now().Format("15:04:05"))
	}
	
	// 如果消息中包含询问日期
	if strings.Contains(userMessage, "日期") || strings.Contains(userMessage, "今天") {
		return fmt.Sprintf("今天是 %s。请注意，这是一个模拟响应，仅用于测试。", time.Now().Format("2006年01月02日"))
	}
	
	// 如果消息中包含天气
	if strings.Contains(userMessage, "天气") {
		return "今天天气晴朗，温度约25°C左右。不过请注意，这只是一个模拟的天气回答，不是真实的天气信息。"
	}
	
	// 默认响应
	return "这是一个模拟的响应，用于测试语音交互系统。在实际部署中，这里会返回真实的AI助手生成的内容。你的消息已收到，但我只能提供有限的预设回答。"
}

// splitIntoChunks 将字符串分割成多个小块
func splitIntoChunks(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		return []string{text}
	}
	
	runes := []rune(text)
	chunks := make([]string, 0, (len(runes)+chunkSize-1)/chunkSize)
	
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	
	return chunks
}

// truncateString 截断字符串，超过最大长度时添加省略号
func truncateString(s string, maxLength int) string {
	runes := []rune(s)
	if len(runes) <= maxLength {
		return s
	}
	return string(runes[:maxLength]) + "..."
} 