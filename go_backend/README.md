# 小智 ESP32 服务器 - Go后端

这个Go后端服务为ESP32设备提供WebSocket通信服务，并通过MQTT进行消息转发。

## 项目结构

```
go_backend/
├── cmd/
│   └── server/
│       └── main.go             # 主程序入口点，启动服务器
├── internal/
│   ├── config/
│   │   └── config.go           # 配置加载逻辑
│   ├── models/
│   │   └── messages.go         # 消息模型定义
│   ├── handlers/
│   │   └── websocket.go        # WebSocket HTTP 处理器
│   ├── conversation/
│   │   └── state.go            # 会话状态管理
│   ├── mqtt/
│   │   └── client.go           # MQTT 客户端连接和操作
│   └── tts/                    # 未来的文本转语音功能
└── go.mod, go.sum
```

## 主要功能

### WebSocket 处理
- 处理从ESP32设备发来的WebSocket连接请求
- 维护活跃的WebSocket连接
- 处理WebSocket消息（文本和二进制音频数据）

### 会话管理
- 管理设备的会话状态（空闲、监听、思考、说话）
- 处理音频数据流并检测沉默
- 生成响应消息

### MQTT集成
- 连接到MQTT代理
- 将WebSocket消息转发到MQTT主题
- 从MQTT接收消息并转发给WebSocket客户端

## 运行方式

```bash
cd go_backend
go run ./cmd/server
```

服务器默认监听端口8000（可通过配置更改）。如果有管理员权限，也会尝试监听端口80以提高与ESP32设备的兼容性。

## API端点

- `/xiaozhi/v1/` - WebSocket连接点
- `/health` - 健康检查端点
- `/status` - 服务器状态信息，返回活跃连接数等信息

## 消息格式

服务器和客户端之间通过JSON格式的消息进行通信，主要消息类型包括：

- `server_hello` - 服务器发送的欢迎消息
- `hello` - 客户端发送的能力声明
- `listen` - 监听状态变化通知
- `listening_start` - 开始监听通知
- `listening_stop` - 停止监听通知
- `spectrogram` - 声谱图数据

二进制音频数据直接通过WebSocket二进制帧传输。 