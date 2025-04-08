package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// DoubanTTSProvider 实现豆包 TTS API 的提供商
type DoubanTTSProvider struct {
	apiKey      string
	apiEndpoint string
	httpClient  *http.Client
	initialized bool
}

// NewDoubanTTSProvider 创建一个新的豆包 TTS 提供商
func NewDoubanTTSProvider(apiKey string) *DoubanTTSProvider {
	return &DoubanTTSProvider{
		apiKey:      apiKey,
		apiEndpoint: "https://api.doubao.com/v1/audio/tts/generation",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SynthesizeSpeech 使用豆包 API 将文本转换为语音
func (p *DoubanTTSProvider) SynthesizeSpeech(text string, options map[string]string) ([]byte, error) {
	if !p.initialized {
		return nil, ErrNotInitialized
	}

	// 默认声音 ID，如果未在选项中指定
	voiceID := "zh_female_qingxin"
	if id, exists := options["voice_id"]; exists && id != "" {
		voiceID = id
	}

	// 默认输出格式为 MP3
	format := "mp3"
	if fmt, exists := options["format"]; exists && fmt != "" {
		format = fmt
	}

	// 准备请求体
	requestBody := map[string]interface{}{
		"model":       "doubao-tts-v1",
		"input":       text,
		"voice":       voiceID,
		"format":      format,
		"speed":       1.0, // 默认速度
		"temperature": 0.3, // 默认温度
	}

	// 如果选项中提供了速度，则使用该值
	if speed, exists := options["speed"]; exists {
		var speedFloat float64
		fmt.Sscanf(speed, "%f", &speedFloat)
		requestBody["speed"] = speedFloat
	}

	// 将请求体转换为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", p.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "application/octet-stream")

	// 发送请求
	log.Printf("[TTS:Douban] Sending TTS request for text: %s", text)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from API [%d]: %s", resp.StatusCode, string(body))
	}

	// 读取响应体
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	log.Printf("[TTS:Douban] Successfully generated audio, size: %d bytes", len(audioData))
	return audioData, nil
}

// GetVoices 返回豆包 TTS 提供的可用声音
func (p *DoubanTTSProvider) GetVoices() ([]Voice, error) {
	if !p.initialized {
		return nil, ErrNotInitialized
	}

	// 返回豆包支持的声音列表
	// 注意：这是硬编码的，实际应用中可能需要调用 API 获取最新支持的声音
	return []Voice{
		{
			ID:       "zh_female_qingxin",
			Name:     "清新女声",
			Gender:   "female",
			Language: "zh-CN",
			Tags: map[string]string{
				"type": "neural",
				"age":  "young",
			},
		},
		{
			ID:       "zh_male_wenzhong",
			Name:     "稳重男声",
			Gender:   "male",
			Language: "zh-CN",
			Tags: map[string]string{
				"type": "neural",
				"age":  "adult",
			},
		},
		{
			ID:       "zh_female_wenzhong",
			Name:     "稳重女声",
			Gender:   "female",
			Language: "zh-CN",
			Tags: map[string]string{
				"type": "neural",
				"age":  "adult",
			},
		},
	}, nil
}

// Initialize 初始化豆包 TTS 提供商
func (p *DoubanTTSProvider) Initialize() error {
	if p.apiKey == "" {
		return fmt.Errorf("douban TTS API key is required")
	}

	log.Printf("[TTS:Douban] Initializing Douban TTS provider")
	p.initialized = true
	return nil
}

// Cleanup 清理豆包 TTS 提供商资源
func (p *DoubanTTSProvider) Cleanup() error {
	log.Printf("[TTS:Douban] Cleaning up Douban TTS provider")
	p.initialized = false
	return nil
} 