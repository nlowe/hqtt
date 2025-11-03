package mqtt

import (
	"context"
	"log/slog"
)

// Subscription holds metadata for a MQTT subscription for a given topic. It implements fmt.Stringer and slog.LogValuer.
type Subscription struct {
	Topic   string
	Options ReadOptions
}

func (s Subscription) String() string {
	return s.Topic
}

func (s Subscription) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("topic", s.Topic),
		slog.Any("options", s.Options),
	)
}

// Handler is the MQTT equivalent to http.Handler. It is a callback configured for an MQTT Subscription.
//
// Because a handler may receive a message at any time, they do not directly return errors. Implementations should
// provide a way to deal with errors separately. Handlers must not block. Any long-running operations should be run from
// a new goroutine started by the Handler instead.
//
// If the handler needs to write any response message to MQTT, it should use the provided writer and return. It is not
// valid to use Writer or message slice after returning.
type Handler interface {
	ServeMQTT(w Writer, topic string, message []byte)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as MQTT handlers. If f is a function with
// the appropriate signature, HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(Writer, string, []byte)

func (f HandlerFunc) ServeMQTT(w Writer, topic string, message []byte) {
	f(w, topic, message)
}

// Subscriber manages MQTT Subscriptions
type Subscriber interface {
	// Subscribe configures the underlying MQTT connection to send the client messages for the provided subscriptions.
	// The provided Handler will be called for all subscribed topics in this call.
	Subscribe(ctx context.Context, handler Handler, subscriptions ...Subscription) error

	// Unsubscribe removes any subscriptions configured for the specified topics.
	Unsubscribe(ctx context.Context, topics ...string) error
}
