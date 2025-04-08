package tts

import (
	"log"
	"sync"
)

// Provider 表示不同的TTS供应商接口
type Provider interface {
	// SynthesizeSpeech 将文本转换为音频字节
	SynthesizeSpeech(text string, options map[string]string) ([]byte, error)
	
	// GetVoices 返回可用的声音列表
	GetVoices() ([]Voice, error)
	
	// Initialize 初始化TTS服务提供商
	Initialize() error
	
	// Cleanup 清理资源
	Cleanup() error
}

// Voice 表示一个TTS声音
type Voice struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Gender   string            `json:"gender"`
	Language string            `json:"language"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// TTSManager 管理多个TTS提供商
type TTSManager struct {
	providers  map[string]Provider
	mutex      sync.RWMutex
	initialized bool
	defaultProvider string
}

// NewTTSManager 创建一个新的TTS管理器
func NewTTSManager() *TTSManager {
	return &TTSManager{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider 注册一个TTS提供商
func (tm *TTSManager) RegisterProvider(name string, provider Provider) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	tm.providers[name] = provider
	
	// 如果这是第一个注册的提供商，将其设为默认
	if tm.defaultProvider == "" {
		tm.defaultProvider = name
	}
	
	log.Printf("[TTS] Registered provider: %s", name)
}

// SetDefaultProvider 设置默认的TTS提供商
func (tm *TTSManager) SetDefaultProvider(name string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	if _, exists := tm.providers[name]; !exists {
		return ErrProviderNotFound
	}
	
	tm.defaultProvider = name
	return nil
}

// Initialize 初始化所有TTS提供商
func (tm *TTSManager) Initialize() error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	for name, provider := range tm.providers {
		if err := provider.Initialize(); err != nil {
			log.Printf("[TTS] Failed to initialize provider %s: %v", name, err)
			return err
		}
	}
	
	tm.initialized = true
	return nil
}

// GetProvider 获取指定的 TTS 提供商
func (tm *TTSManager) GetProvider(name string) (Provider, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	// 如果没有指定名称，使用默认提供商
	if name == "" {
		name = tm.defaultProvider
	}
	
	provider, exists := tm.providers[name]
	if !exists {
		return nil, ErrProviderNotFound
	}
	
	return provider, nil
}

// SynthesizeSpeech 使用默认提供商合成语音
func (tm *TTSManager) SynthesizeSpeech(text string, options map[string]string) ([]byte, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	if !tm.initialized {
		return nil, ErrNotInitialized
	}
	
	provider, exists := tm.providers[tm.defaultProvider]
	if !exists {
		return nil, ErrProviderNotFound
	}
	
	return provider.SynthesizeSpeech(text, options)
}

// 错误定义
var (
	ErrProviderNotFound = NewTTSError("tts provider not found")
	ErrNotInitialized   = NewTTSError("tts manager not initialized")
)

// TTSError 表示TTS操作中的错误
type TTSError struct {
	Message string
}

// NewTTSError 创建一个新的TTS错误
func NewTTSError(message string) *TTSError {
	return &TTSError{Message: message}
}

// Error 实现error接口
func (e *TTSError) Error() string {
	return e.Message
} 