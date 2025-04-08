package llm

import (
	"log"
	"sync"
)

// Provider 表示不同的 LLM (大语言模型) 提供商接口
type Provider interface {
	// Chat 发送对话消息并获取响应
	Chat(messages []Message, options map[string]interface{}) (*Response, error)
	
	// StreamChat 发送对话消息并通过流式接口获取响应
	StreamChat(messages []Message, options map[string]interface{}, callback StreamCallback) error
	
	// Initialize 初始化 LLM 服务提供商
	Initialize() error
	
	// Cleanup 清理资源
	Cleanup() error
}

// Message 表示一条对话消息
type Message struct {
	Role    string                 `json:"role"`     // 角色：user, assistant, system
	Content string                 `json:"content"`  // 消息内容
	Metadata map[string]interface{} `json:"metadata,omitempty"` // 元数据
}

// Response 表示 LLM 的响应
type Response struct {
	Content     string                 `json:"content"`      // 响应内容
	FinishReason string                 `json:"finish_reason"` // 结束原因
	Metadata    map[string]interface{} `json:"metadata,omitempty"`  // 元数据
}

// StreamCallback 是处理流式响应的回调函数类型
type StreamCallback func(chunk *ResponseChunk) error

// ResponseChunk 表示流式响应的一个数据块
type ResponseChunk struct {
	Content     string                 `json:"content"`      // 当前块内容
	FinishReason string                 `json:"finish_reason,omitempty"` // 结束原因（仅在最后一个块中存在）
	IsFinal     bool                   `json:"is_final"`     // 是否为最后一个块
	Metadata    map[string]interface{} `json:"metadata,omitempty"`  // 元数据
}

// LLMManager 管理多个 LLM 提供商
type LLMManager struct {
	providers  map[string]Provider
	mutex      sync.RWMutex
	initialized bool
	defaultProvider string
}

// NewLLMManager 创建一个新的 LLM 管理器
func NewLLMManager() *LLMManager {
	return &LLMManager{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider 注册一个 LLM 提供商
func (lm *LLMManager) RegisterProvider(name string, provider Provider) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	
	lm.providers[name] = provider
	
	// 如果这是第一个注册的提供商，将其设为默认
	if lm.defaultProvider == "" {
		lm.defaultProvider = name
	}
	
	log.Printf("[LLM] Registered provider: %s", name)
}

// SetDefaultProvider 设置默认的 LLM 提供商
func (lm *LLMManager) SetDefaultProvider(name string) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	
	if _, exists := lm.providers[name]; !exists {
		return ErrProviderNotFound
	}
	
	lm.defaultProvider = name
	return nil
}

// Initialize 初始化所有 LLM 提供商
func (lm *LLMManager) Initialize() error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()
	
	for name, provider := range lm.providers {
		if err := provider.Initialize(); err != nil {
			log.Printf("[LLM] Failed to initialize provider %s: %v", name, err)
			return err
		}
	}
	
	lm.initialized = true
	return nil
}

// Chat 使用默认提供商进行对话
func (lm *LLMManager) Chat(messages []Message, options map[string]interface{}) (*Response, error) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()
	
	if !lm.initialized {
		return nil, ErrNotInitialized
	}
	
	provider, exists := lm.providers[lm.defaultProvider]
	if !exists {
		return nil, ErrProviderNotFound
	}
	
	return provider.Chat(messages, options)
}

// StreamChat 使用默认提供商进行流式对话
func (lm *LLMManager) StreamChat(messages []Message, options map[string]interface{}, callback StreamCallback) error {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()
	
	if !lm.initialized {
		return ErrNotInitialized
	}
	
	provider, exists := lm.providers[lm.defaultProvider]
	if !exists {
		return ErrProviderNotFound
	}
	
	return provider.StreamChat(messages, options, callback)
}

// GetProvider 获取指定的 LLM 提供商
func (lm *LLMManager) GetProvider(name string) (Provider, error) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()
	
	provider, exists := lm.providers[name]
	if !exists {
		return nil, ErrProviderNotFound
	}
	
	return provider, nil
}

// 错误定义
var (
	ErrProviderNotFound = NewLLMError("llm provider not found")
	ErrNotInitialized   = NewLLMError("llm manager not initialized")
)

// LLMError 表示 LLM 操作中的错误
type LLMError struct {
	Message string
}

// NewLLMError 创建一个新的 LLM 错误
func NewLLMError(message string) *LLMError {
	return &LLMError{Message: message}
}

// Error 实现 error 接口
func (e *LLMError) Error() string {
	return e.Message
} 