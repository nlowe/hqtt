package testqtt

import (
	"context"
	"sync"
	"uuid"

	"github.com/eclipse/paho.golang/paho"

	"github.com/nlowe/hqtt/mqtt"
)

// Client implements mqtt.Subscriber and mqtt.Writer
type Client struct {
	sync.RWMutex
	Seen, Sent *Recorder

	id uuid.UUID
	b  *Broker
	r  paho.Router

	rejectWrite, rejectSubscribe, rejectUnsubscribe error
}

var _ mqtt.Subscriber = &Client{}
var _ mqtt.Writer = &Client{}

// RejectSubscribeWith configures this Client to return the provided error for all new calls to Subscribe. Provide a nil
// error to clear this condition.
func (c *Client) RejectSubscribeWith(err error) {
	c.Lock()
	defer c.Unlock()

	c.rejectSubscribe = err
}

// Subscribe configures one or more subscriptions to route to the provided mqtt.Handler. The handler will be invoked
// with this Client passed as the mqtt.Writer argument.
func (c *Client) Subscribe(_ context.Context, handler mqtt.Handler, subscriptions ...mqtt.Subscription) error {
	c.Lock()
	defer c.Unlock()

	if c.rejectSubscribe != nil {
		return c.rejectSubscribe
	}

	for _, sub := range subscriptions {
		c.r.RegisterHandler(sub.Topic, func(publish *paho.Publish) {
			c.Seen.Record(Message{
				Topic: publish.Topic,
				Opts: mqtt.WriteOptions{
					QoS:    mqtt.QualityOfService(publish.QoS),
					Retain: publish.Retain,
				},
				Data: publish.Payload,
			})
			handler.ServeMQTT(c, publish.Topic, publish.Payload)
		})
	}

	return nil
}

// RejectUnsubscribeWith configures this Client to return the provided error for all new calls to Unsubscribe. Provide a
// nil error to clear this condition.
func (c *Client) RejectUnsubscribeWith(err error) {
	c.Lock()
	defer c.Unlock()

	c.rejectUnsubscribe = err
}

// Unsubscribe clears all registered mqtt.Handler callbacks for the provided topics.
func (c *Client) Unsubscribe(_ context.Context, topics ...string) error {
	c.Lock()
	defer c.Unlock()

	if c.rejectUnsubscribe != nil {
		return c.rejectUnsubscribe
	}

	for _, topic := range topics {
		c.r.UnregisterHandler(topic)
	}

	return nil
}

// RejectWriteWith configures this Client to return the provided error for all new calls to WriteTopic. Provide a nil
// error to clear this condition.
func (c *Client) RejectWriteWith(err error) {
	c.Lock()
	defer c.Unlock()

	c.rejectWrite = err
}

// WriteTopic publishes the provided message on the provided topic. The Broker this Client was created from will invoke
// handlers on all Client objects in a new goroutine for each Client.
func (c *Client) WriteTopic(ctx context.Context, topic string, options mqtt.WriteOptions, value []byte) error {
	c.RLock()
	defer c.RUnlock()

	if c.rejectWrite != nil {
		return c.rejectWrite
	}

	c.Sent.Record(Message{
		Topic: topic,
		Opts:  options,
		Data:  value,
	})
	return c.b.WriteTopic(ctx, topic, options, value)
}
