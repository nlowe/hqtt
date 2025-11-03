module example

go 1.25

replace github.com/nlowe/hqtt => ../

require (
	github.com/eclipse/paho.golang v0.23.0
	github.com/nlowe/hqtt v0.0.0-00010101000000-000000000000
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.43.0 // indirect
)
