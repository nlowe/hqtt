package autopaho

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	// TODO: Can we pull this out easily and make this an optional dependency without making the module too complicated?
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	hqttlog "github.com/nlowe/hqtt/log"
	"github.com/nlowe/hqtt/mqtt"
)

type adapter struct {
	mu sync.Mutex

	conn *autopaho.ConnectionManager
	r    paho.Router

	subscriptions map[string]paho.SubscribeOptions

	log *slog.Logger
}

var _ mqtt.Writer = &adapter{}
var _ mqtt.Subscriber = &adapter{}

func DialMQTT(ctx context.Context, config autopaho.ClientConfig) (mqtt.Writer, mqtt.Subscriber, func(ctx context.Context) error, error) {
	a := &adapter{
		r: paho.NewStandardRouter(),

		subscriptions: map[string]paho.SubscribeOptions{},

		log: hqttlog.ForComponent("autopaho"),
	}

	// Overwrite the OnConnectionUp handler to deal with re-subscribing.
	originalOnConnUp := config.OnConnectionUp
	config.OnConnectionUp = func(manager *autopaho.ConnectionManager, connack *paho.Connack) {
		a.onReconnect(ctx)

		if originalOnConnUp != nil {
			originalOnConnUp(manager, connack)
		}
	}

	// Lock the adapter before starting the connection so the first OnConnectionUp callback (which calls a.onReconnect)
	// blocks until after a.conn is assigned.
	a.mu.Lock()
	a.log.Info("Connecting to mqtt broker")
	conn, err := autopaho.NewConnection(ctx, config)
	if err != nil {
		a.mu.Unlock()
		return nil, nil, nil, err
	}

	a.conn = conn
	a.mu.Unlock()

	a.log.Debug("Waiting for connection to be ready")
	if err = conn.AwaitConnection(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("mqtt: wait for connection: %w", err)
	}

	a.log.Debug("Connected to mqtt broker")
	conn.AddOnPublishReceived(func(rx autopaho.PublishReceived) (bool, error) {
		a.r.Route(rx.Packet.Packet())
		return true, nil
	})

	return a, a, conn.Disconnect, nil
}

func (a *adapter) onReconnect(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.subscriptions) == 0 {
		return
	}

	sub := &paho.Subscribe{
		Subscriptions: make([]paho.SubscribeOptions, 0, len(a.subscriptions)),
	}

	for _, s := range a.subscriptions {
		sub.Subscriptions = append(sub.Subscriptions, s)
	}

	a.log.Debug("Reconnected to MQTT. Re-sending subscriptions.")
	_, err := a.conn.Subscribe(ctx, sub)
	if err != nil {
		// TODO: Retry? Somehow lift this failure to the consumer?
		a.log.With(hqttlog.Error(err)).Error("Failed to re-subscribe to mqtt topics")
	}
}

func (a *adapter) WriteTopic(ctx context.Context, topic string, options mqtt.WriteOptions, value []byte) error {
	a.log.With(slog.String("topic", topic), slog.Any("options", options), slog.String("payload", string(value))).Debug("Publishing payload")

	_, err := a.conn.Publish(ctx, &paho.Publish{
		QoS:     uint8(options.QoS),
		Retain:  options.Retain,
		Topic:   topic,
		Payload: value,
	})

	return err
}

func (a *adapter) Subscribe(ctx context.Context, handler mqtt.Handler, subscriptions ...mqtt.Subscription) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(subscriptions) == 0 {
		return nil
	}

	sub := &paho.Subscribe{
		Subscriptions: make([]paho.SubscribeOptions, len(subscriptions)),
	}

	for i, s := range subscriptions {
		opts := paho.SubscribeOptions{
			Topic:             s.Topic,
			QoS:               uint8(s.Options.QoS),
			RetainHandling:    uint8(s.Options.RetainHandling),
			NoLocal:           s.Options.NoLocal,
			RetainAsPublished: s.Options.RetainAsPublished,
		}

		a.subscriptions[s.Topic] = opts
		sub.Subscriptions[i] = opts

		a.r.RegisterHandler(s.Topic, func(publish *paho.Publish) {
			handler.ServeMQTT(a, publish.Topic, publish.Payload)
		})
	}

	a.log.With(slog.Any("subscriptions", subscriptions)).Debug("Subscribing to MQTT Topic(s)")
	_, err := a.conn.Subscribe(ctx, sub)
	return err
}

func (a *adapter) Unsubscribe(ctx context.Context, topics ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, t := range topics {
		delete(a.subscriptions, t)
	}

	a.log.With(slog.Any("topics", topics)).Debug("Unsubscribing from MQTT Topic(s)")
	_, err := a.conn.Unsubscribe(ctx, &paho.Unsubscribe{
		Topics: topics,
	})

	return err
}
