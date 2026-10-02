package testqtt

import (
	"context"
	"sync"
	"uuid"

	"github.com/eclipse/paho.golang/packets"
	"github.com/eclipse/paho.golang/paho"

	"github.com/nlowe/hqtt/mqtt"
)

type Message struct {
	Topic string
	Opts  mqtt.WriteOptions
	Data  []byte
}

// Broker implements an in-memory MQTT broker for testing. No guarantees are made about the order messages are processed
// or the order subscriptions are configured.
type Broker struct {
	sync.RWMutex
	Recorder

	clients map[uuid.UUID]*Client
}

// Connect creates an in-memory MQTT Client associated with this Broker. The returned Client implements mqtt.Writer and
// mqtt.Subscriber.
func (b *Broker) Connect() *Client {
	result := &Client{
		id: uuid.New(),
		b:  b,

		Seen: &Recorder{},
		Sent: &Recorder{},
		r:    paho.NewStandardRouter(),
	}

	b.Lock()
	defer b.Unlock()

	if b.clients == nil {
		b.clients = map[uuid.UUID]*Client{}
	}

	b.clients[result.id] = result
	return result
}

// Disconnect removes the provided client from this broker. It will no longer receive messages.
func (b *Broker) Disconnect(c *Client) {
	b.Lock()
	defer b.Unlock()

	if b.clients == nil {
		return
	}

	delete(b.clients, c.id)
}

// WriteTopic publishes the provided message to all clients associated with this Broker.
func (b *Broker) WriteTopic(_ context.Context, topic string, options mqtt.WriteOptions, value []byte) error {
	b.Lock()
	defer b.Unlock()

	if b.clients == nil {
		return nil
	}

	for _, c := range b.clients {
		c.r.Route(&packets.Publish{
			Payload:    value,
			Topic:      topic,
			Properties: &packets.Properties{},
			//PacketID:   0,
			QoS:    byte(options.QoS),
			Retain: options.Retain,
		})
	}

	b.Record(Message{
		Topic: topic,
		Opts:  options,
		Data:  value,
	})
	return nil
}
