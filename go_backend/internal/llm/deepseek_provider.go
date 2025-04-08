package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// DeepseekProvider 实现 Deepseek API
type DeepseekProvider struct {
	apiKey      string
	apiEndpoint string
	model       string
	httpClient  *http.Client
	initialized bool
}

// DeepseekRequestBody 表示发送到 Deepseek API 的请求体结构
type DeepseekRequestBody struct {
	Model       string                 `json:"model"`
	Messages    []DeepseekMessage      `json:"messages"`
	Stream      bool                   `json:"stream"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	FrequencyP  float64                `json:"frequency_penalty,omitempty"`
	PresenceP   float64                `json:"presence_penalty,omitempty"`
	Tools       []interface{}          `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
	System      string                 `json:"system,omitempty"`
}

// DeepseekMessage 表示 Deepseek API 消息格式
type DeepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepseekResponse 表示 Deepseek API 的响应结构
type DeepseekResponse struct {
	ID                string                `json:"id"`
	Object            string                `json:"object"`
	Created           int64                 `json:"created"`
	Model             string                `json:"model"`
	Choices           []DeepseekChoice      `json:"choices"`
	Usage             DeepseekUsage         `json:"usage"`
}

// DeepseekChoice 表示 Deepseek API 的响应选择
type DeepseekChoice struct {
	Index        int             `json:"index"`
	FinishReason string          `json:"finish_reason"`
	Message      DeepseekMessage `json:"message"`
}

// DeepseekUsage 表示 Deepseek API 的使用统计
type DeepseekUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// DeepseekStreamResponse 表示 Deepseek API 的流式响应
type DeepseekStreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice 表示 Deepseek API 的流式响应选择
type StreamChoice struct {
	Index        int                `json:"index"`
	Delta        StreamDelta        `json:"delta"`
	FinishReason string             `json:"finish_reason"`
}

// StreamDelta 表示 Deepseek API 的流式响应增量内容
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// NewDeepseekProvider 创建一个新的 Deepseek 提供商
func NewDeepseekProvider(apiKey string, model string) *DeepseekProvider {
	if model == "" {
		model = "deepseek-chat" // 默认模型
	}
	
	return &DeepseekProvider{
		apiKey:      apiKey,
		apiEndpoint: "https://api.deepseek.com/v1/chat/completions",
		model:       model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

// Chat 实现非流式对话
func (p *DeepseekProvider) Chat(messages []Message, options map[string]interface{}) (*Response, error) {
	if !p.initialized {
		return nil, ErrNotInitialized
	}
	
	// 准备请求
	dsMessages := convertToDeepseekMessages(messages)
	
	body := DeepseekRequestBody{
		Model:       p.model,
		Messages:    dsMessages,
		Stream:      false,
		Temperature: 0.7,  // 默认值
		MaxTokens:   2000, // 默认值
	}
	
	// 处理选项
	p.applyOptions(&body, options)
	
	// 发送请求并获取响应
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}
	
	req, err := http.NewRequest("POST", p.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	
	log.Printf("[LLM:Deepseek] Sending Chat request with %d messages", len(messages))
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from API [%d]: %s", resp.StatusCode, string(bodyBytes))
	}
	
	// 解析响应
	var deepseekResp DeepseekResponse
	if err := json.NewDecoder(resp.Body).Decode(&deepseekResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	// 检查是否有选择
	if len(deepseekResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	
	// 转换为通用响应格式
	result := &Response{
		Content:     deepseekResp.Choices[0].Message.Content,
		FinishReason: deepseekResp.Choices[0].FinishReason,
		Metadata: map[string]interface{}{
			"model":       deepseekResp.Model,
			"usage":       deepseekResp.Usage,
			"id":          deepseekResp.ID,
			"created":     deepseekResp.Created,
		},
	}
	
	return result, nil
}

// StreamChat 实现流式对话
func (p *DeepseekProvider) StreamChat(messages []Message, options map[string]interface{}, callback StreamCallback) error {
	if !p.initialized {
		return ErrNotInitialized
	}
	
	// 准备请求
	dsMessages := convertToDeepseekMessages(messages)
	
	body := DeepseekRequestBody{
		Model:       p.model,
		Messages:    dsMessages,
		Stream:      true,
		Temperature: 0.7,  // 默认值
		MaxTokens:   2000, // 默认值
	}
	
	// 处理选项
	p.applyOptions(&body, options)
	
	// 发送请求
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}
	
	req, err := http.NewRequest("POST", p.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	
	log.Printf("[LLM:Deepseek] Sending StreamChat request with %d messages", len(messages))
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response from API [%d]: %s", resp.StatusCode, string(bodyBytes))
	}
	
	// 处理 Server-Sent Events (SSE) 流
	reader := bufio.NewReader(resp.Body)
	contentBuilder := strings.Builder{}
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}
		
		line = strings.TrimSpace(line)
		
		// 跳过空行或 SSE 注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		
		// 处理数据行
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			
			// 检查是否是流结束标记
			if data == "[DONE]" {
				// 创建最终响应块
				finalChunk := &ResponseChunk{
					Content:     contentBuilder.String(),
					IsFinal:     true,
					FinishReason: "stop", // 假设正常结束
				}
				
				// 调用回调处理最终块
				if err := callback(finalChunk); err != nil {
					return fmt.Errorf("error in callback (final): %w", err)
				}
				break
			}
			
			// 解析 JSON 响应
			var streamResp DeepseekStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				return fmt.Errorf("error parsing stream data: %w", err)
			}
			
			// 确保有选择
			if len(streamResp.Choices) == 0 {
				continue
			}
			
			choice := streamResp.Choices[0]
			content := choice.Delta.Content
			
			// 添加到内容构建器
			contentBuilder.WriteString(content)
			
			// 创建响应块
			chunk := &ResponseChunk{
				Content: content,
				IsFinal: false,
			}
			
			// 如果有结束原因，设置相应字段
			if choice.FinishReason != "" {
				chunk.FinishReason = choice.FinishReason
				chunk.IsFinal = true
			}
			
			// 调用回调处理块
			if err := callback(chunk); err != nil {
				return fmt.Errorf("error in callback: %w", err)
			}
		}
	}
	
	return nil
}

// Initialize 初始化 Deepseek 提供商
func (p *DeepseekProvider) Initialize() error {
	if p.apiKey == "" {
		return fmt.Errorf("Deepseek API key is required")
	}
	
	log.Printf("[LLM:Deepseek] Initializing Deepseek provider with model: %s", p.model)
	p.initialized = true
	return nil
}

// Cleanup 清理 Deepseek 提供商资源
func (p *DeepseekProvider) Cleanup() error {
	log.Printf("[LLM:Deepseek] Cleaning up Deepseek provider")
	p.initialized = false
	return nil
}

// applyOptions 应用选项到请求体
func (p *DeepseekProvider) applyOptions(body *DeepseekRequestBody, options map[string]interface{}) {
	// 应用温度
	if temp, ok := options["temperature"].(float64); ok {
		body.Temperature = temp
	}
	
	// 应用最大令牌数
	if maxTokens, ok := options["max_tokens"].(int); ok {
		body.MaxTokens = maxTokens
	}
	
	// 应用 top_p
	if topP, ok := options["top_p"].(float64); ok {
		body.TopP = topP
	}
	
	// 应用频率惩罚
	if freqP, ok := options["frequency_penalty"].(float64); ok {
		body.FrequencyP = freqP
	}
	
	// 应用存在惩罚
	if presP, ok := options["presence_penalty"].(float64); ok {
		body.PresenceP = presP
	}
	
	// 应用系统提示
	if system, ok := options["system"].(string); ok {
		body.System = system
	} else {
		// 查找并提取系统消息
		for _, msg := range body.Messages {
			if msg.Role == "system" {
				body.System = msg.Content
				break
			}
		}
	}
	
	// 过滤掉系统消息（因为现在用 system 字段）
	filteredMessages := make([]DeepseekMessage, 0, len(body.Messages))
	for _, msg := range body.Messages {
		if msg.Role != "system" {
			filteredMessages = append(filteredMessages, msg)
		}
	}
	body.Messages = filteredMessages
}

// convertToDeepseekMessages 将通用消息格式转换为 Deepseek 消息格式
func convertToDeepseekMessages(messages []Message) []DeepseekMessage {
	result := make([]DeepseekMessage, 0, len(messages))
	
	for _, msg := range messages {
		result = append(result, DeepseekMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	return result
} 