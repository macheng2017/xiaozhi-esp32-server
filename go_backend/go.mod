module github.com/xiaozhi-esp32-server/go_backend

go 1.24.2

replace github.com/xiaozhi-esp32-server/go_backend => ./

require (
	github.com/eclipse/paho.mqtt.golang v1.5.0
	github.com/gorilla/websocket v1.5.3
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
)
