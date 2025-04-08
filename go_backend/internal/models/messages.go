package models

// MessageType 定义了消息类型常量
type MessageType string

// 定义各种消息类型常量
const (
	TypeServerHello    MessageType = "hello"
	TypeHello          MessageType = "hello"
	TypeListen         MessageType = "listen"
	TypeListeningStart MessageType = "listening_start"
	TypeListeningStop  MessageType = "listening_stop"
	TypeSpectrogram    MessageType = "spectrogram"
	TypeText           MessageType = "text"
	TypeTTS            MessageType = "tts"
	TypeError          MessageType = "error"
)

// ListenState 定义会话中的状态类型
type ListenState string

// 定义会话状态常量
const (
	StateIdle      ListenState = "idle"
	StateListening ListenState = "listening"
	StateThinking  ListenState = "thinking"
	StateSpeaking  ListenState = "speaking"
)

// ServerHelloMessage 是服务器发送的欢迎消息
type ServerHelloMessage struct {
	Type      MessageType `json:"type"`
	Status    string      `json:"status"`
	Transport string      `json:"transport"`
	SessionID string      `json:"session_id"`
}

// ClientHelloMessage 是客户端发送的能力声明消息
type ClientHelloMessage struct {
	Type        MessageType `json:"type"`
	Version     int         `json:"version"`
	Transport   string      `json:"transport"`
	AudioParams AudioParams `json:"audio_params"`
}

// AudioParams 定义了音频参数
type AudioParams struct {
	Format        string `json:"format"`
	SampleRate    int    `json:"sample_rate"`
	Channels      int    `json:"channels"`
	FrameDuration int    `json:"frame_duration"`
}

// ListenMessage 定义了各种状态变化的消息
type ListenMessage struct {
	Type  MessageType `json:"type"`
	State ListenState `json:"state,omitempty"`
	Text  string      `json:"text,omitempty"`
}

// DeviceCapability 表示设备的能力 (如音量控制、灯光控制等)
type DeviceCapability struct {
	Type    string      `json:"type"`
	Name    string      `json:"name"`
	Options interface{} `json:"options,omitempty"`
}

// TextMessage 定义了文本消息
type TextMessage struct {
	Type MessageType `json:"type"`
	Text string      `json:"text"`
}

// ErrorMessage 定义了错误消息
type ErrorMessage struct {
	Type    MessageType `json:"type"`
	Error   string      `json:"error"`
	Code    int         `json:"code,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// TTSMessage 定义了文本转语音请求消息
type TTSMessage struct {
	Type    MessageType         `json:"type"`
	Text    string              `json:"text"`
	Options map[string]string   `json:"options,omitempty"`
}

// SimpleMessage 定义了没有附加数据的简单消息
type SimpleMessage struct {
	Type MessageType `json:"type"`
} 