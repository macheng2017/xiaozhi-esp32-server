package tts

import (
	"log"
	"math/rand"
)

// MockProvider 是一个简单的模拟TTS提供商，用于开发和测试
type MockProvider struct {
	initialized bool
}

// NewMockProvider 创建一个新的模拟TTS提供商
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// SynthesizeSpeech 模拟文本转语音过程，返回随机生成的"音频"数据
func (p *MockProvider) SynthesizeSpeech(text string, options map[string]string) ([]byte, error) {
	if !p.initialized {
		return nil, ErrNotInitialized
	}
	
	log.Printf("[TTS:Mock] Synthesizing speech for text: %s", text)
	
	// 模拟处理时间（这里只是简单记录）
	log.Printf("[TTS:Mock] Speech options: %v", options)
	
	// 生成一些随机字节作为"音频"数据（仅用于测试）
	byteCount := len(text) * 100 // 假设每个字符产生100字节的音频
	audioData := make([]byte, byteCount)
	rand.Read(audioData)
	
	return audioData, nil
}

// GetVoices 返回一组模拟的声音
func (p *MockProvider) GetVoices() ([]Voice, error) {
	if !p.initialized {
		return nil, ErrNotInitialized
	}
	
	// 返回一些模拟的声音
	return []Voice{
		{
			ID:       "mock-female-1",
			Name:     "小美",
			Gender:   "female",
			Language: "zh-CN",
			Tags: map[string]string{
				"type": "neural",
				"age":  "young",
			},
		},
		{
			ID:       "mock-male-1",
			Name:     "小刚",
			Gender:   "male",
			Language: "zh-CN",
			Tags: map[string]string{
				"type": "neural",
				"age":  "adult",
			},
		},
	}, nil
}

// Initialize 初始化模拟TTS提供商
func (p *MockProvider) Initialize() error {
	log.Printf("[TTS:Mock] Initializing mock TTS provider")
	p.initialized = true
	return nil
}

// Cleanup 清理模拟TTS提供商资源
func (p *MockProvider) Cleanup() error {
	log.Printf("[TTS:Mock] Cleaning up mock TTS provider")
	p.initialized = false
	return nil
} 