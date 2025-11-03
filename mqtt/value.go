package mqtt

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nlowe/hqtt/log"
)

var (
	// ErrNoMarshaler is the error returned when a Value does not have an associated ValueMarshaler, which is required
	// to write the value to MQTT.
	ErrNoMarshaler = fmt.Errorf("no marshaler configured")
	// ErrNeverWritten is the error returned by Value.Republish when Value.Write was not previously called successfully.
	ErrNeverWritten = fmt.Errorf("value was never written")
)

// QualityOfService determines what level of guarantee the broker should provide when delivering messages. It implements
// fmt.Stringer and slog.LogValuer.
type QualityOfService uint8

func (q QualityOfService) String() string {
	switch q {
	case QOSAtMostOnce:
		return "at most once (0)"
	case QOSAtLeastOnce:
		return "at least once (1)"
	case QOSExactlyOnce:
		return "exactly once (2)"
	default:
		panic(fmt.Errorf("invalid quality of service value: %d", q))
	}
}

func (q QualityOfService) LogValue() slog.Value {
	return slog.StringValue(q.String())
}

const (
	// QOSAtMostOnce offers "fire and forget" messaging with no acknowledgment from the receiver. This is the default.
	QOSAtMostOnce QualityOfService = iota
	// QOSAtLeastOnce ensures that messages are delivered at least once by requiring a PUBACK acknowledgment.
	QOSAtLeastOnce
	// QOSExactlyOnce guarantees that each message is delivered exactly once by using a four-step handshake (PUBLISH,
	// PUBREC, PUBREL, PUBCOMP).
	QOSExactlyOnce

	// QOSDefault is the default Quality Of Service, QOSAtMostOnce.
	QOSDefault = QOSAtMostOnce
)

// WriteOptions holds options for writing to MQTT. The zero value for WriteOptions uses a QoS of 0 with no retain. It
// implements slog.LogValuer.
type WriteOptions struct {
	// QoS specifies the Quality of Service to use when writing values to MQTT.
	QoS QualityOfService

	// Retain instructs the broker to persist the last message received for a given topic. When a new subscription is
	// created for the topic, the broker will emit this value automatically, whether the publisher is still connected to
	// the broker.
	Retain bool
}

func (w WriteOptions) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("qos", w.QoS),
		slog.Bool("retain", w.Retain),
	)
}

// Value holds a value that can be written to a mqtt topic.
type Value[T any] struct {
	topic string

	marshaler ValueMarshaler[T]
	// TODO: Self-subscribe to get the initial value if retained?
	opts WriteOptions

	mu sync.RWMutex

	v           T
	initialized bool

	log *slog.Logger
}

// NewValue constructs a Value configured for the provided topic and uses the provided marshaler when writing to mqtt
// using default WriteOptions (QoS 0, no retain).
func NewValue[T any](topic string, marshal ValueMarshaler[T]) *Value[T] {
	return NewValueWithOptions(topic, marshal, WriteOptions{})
}

// NewValueWithOptions constructs a Value configured for the provided topic and uses the provided marshaler when writing
// to mqtt using the provided WriteOptions.
func NewValueWithOptions[T any](topic string, marshal ValueMarshaler[T], opts WriteOptions) *Value[T] {
	return &Value[T]{
		topic:     topic,
		marshaler: marshal,
		opts:      opts,

		log: log.ForComponent("mqtt.value"),
	}

}

// FullyQualifiedTopic calculates the MQTT Topic for this value when given the specified prefix. If the underlying Value
// (not the value it holds) is nil, the empty string is returned.
func (v *Value[T]) FullyQualifiedTopic(prefix string) string {
	if v == nil {
		return ""
	}

	return JoinTopic(prefix, v.topic)
}

// Get returns the most recently written value and a bool indicating whether the most recent write was successful, which
// will be false if the value has not yet been written.
func (v *Value[T]) Get() (T, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.v, v.initialized
}

// Republish writes the current value held by this Value to MQTT. Useful if you're not using WriteOptions.Retain and
// need to notify new subscribers of the current state.
func (v *Value[T]) Republish(ctx context.Context, w Writer, prefix string) (T, error) {
	// Copy the value while holding RLock, then release the lock so Write can grab the Lock.
	v.mu.RLock()
	currentValue, initialized := v.v, v.initialized
	v.mu.RUnlock()

	if !initialized {
		return v.v, ErrNeverWritten
	}

	return v.Write(ctx, w, prefix, currentValue)
}

// Write uses the configured marshaler for this value to encode the newValue to the configured topic. It then updates
// the held value. After the call to Write succeeds, future calls to Get will start returning newValue.
func (v *Value[T]) Write(ctx context.Context, w Writer, prefix string, newValue T) (T, error) {
	if v.marshaler == nil {
		return newValue, ErrNoMarshaler
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := v.marshaler(newValue)
	if err != nil {
		return v.v, fmt.Errorf("marshal %+v: %w", newValue, err)
	}

	v.v = newValue
	v.initialized = true
	return v.v, w.WriteTopic(ctx, JoinTopic(prefix, v.topic), v.opts, data)
}

// SubscriptionRetainHandling adjusts how MQTT sends retain values to subscribers. It implements fmt.Stringer and
// slog.LogValuer.
type SubscriptionRetainHandling uint8

func (s SubscriptionRetainHandling) String() string {
	switch s {
	case RetainHandlingSendOnSubscribe:
		return "send on subscribe (0)"
	case RetainHandlingSendOnNewSubscribe:
		return "send on new subscribe (1)"
	case RetainHandlingIgnoreRetained:
		return "ignore retained (2)"
	default:
		panic(fmt.Errorf("invalid subscription retain handling value: %d", s))
	}
}

func (s SubscriptionRetainHandling) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

const (
	// RetainHandlingSendOnSubscribe instructs the broker to send retained messages are whenever a subscription is
	// established, including resubscribe events.
	RetainHandlingSendOnSubscribe SubscriptionRetainHandling = iota
	// RetainHandlingSendOnNewSubscribe instructs the broker to send retained messages are whenever a subscription is
	// newly established (excluding resubscribe events).
	RetainHandlingSendOnNewSubscribe
	// RetainHandlingIgnoreRetained instructs the broker to not send retained messages when a subscription is
	// established.
	RetainHandlingIgnoreRetained

	// RetainHandlingDefault is the default behavior for retaining messages, RetainHandlingSendOnSubscribe.
	RetainHandlingDefault = RetainHandlingSendOnSubscribe
)

// ReadOptions holds options for configuring MQTT Subscriptions. The zero value for ReadOptions uses a QoS of 0 with no
// RetainHandlingDefault. It implements slog.LogValuer.
type ReadOptions struct {
	// QoS specifies the maximum Quality of Service this client supports when setting up subscriptions.
	QoS QualityOfService

	// When true, NoLocal indicates that the server must not forward the message to the client that published it.
	NoLocal bool

	// By default, the retain flag is cleared by the broker when forwarding retained messages. Set RetainAsPublished to
	// true to preserve the Retain flag unchanged when forwarding application messages to subscribers
	RetainAsPublished bool

	RetainHandling SubscriptionRetainHandling
}

func (r ReadOptions) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("qos", r.QoS),
		slog.Bool("no_local", r.NoLocal),
		slog.Bool("retain_as_published", r.RetainAsPublished),
		slog.Any("retain_handling", r.RetainHandling),
	)
}

// RemoteValue holds a value that is populated from a mqtt topic subscription.
type RemoteValue[T any] struct {
	topic       string
	unmarshaler ValueUnmarshaler[T]
	opts        ReadOptions

	mu sync.RWMutex

	watchers []func(T)

	v           T
	initialized bool

	log *slog.Logger
}

// NewRemoteValue constructs a RemoteValue by subscribing to the specified topic on the provided SubscriptionRouter. It
// uses the provided ValueUnmarshaler to decode payloads from mqtt and default ReadOptions (QoS 0,
// RetainHandlingDefault).
func NewRemoteValue[T any](topic string, unmarshaler ValueUnmarshaler[T]) *RemoteValue[T] {
	return NewRemoteValueWithOptions(topic, unmarshaler, ReadOptions{})
}

// NewRemoteValueWithOptions constructs a RemoteValue by subscribing to the specified topic on the provided
// SubscriptionRouter. It uses the provided ValueUnmarshaler to decode payloads from mqtt with the provided ReadOptions.
func NewRemoteValueWithOptions[T any](topic string, unmarshaler ValueUnmarshaler[T], opts ReadOptions) *RemoteValue[T] {
	return &RemoteValue[T]{
		topic:       topic,
		unmarshaler: unmarshaler,
		opts:        opts,

		log: log.ForComponent("mqtt.value.remote").With(slog.String("topic", topic)),
	}
}

// ServeMQTT implements mqtt.Handler for this RemoteValue by unmarshalling a value from the provided payload if the
// topic exactly matches the configured topic for this RemoteValue. It then invokes any watcher callbacks. If
// unmarshalling fails, the watchers are not called and an error is logged. See the log package for details on
// configuring this logger.
func (v *RemoteValue[T]) ServeMQTT(_ Writer, topic string, payload []byte) {
	if v == nil {
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if v.topic != topic {
		return
	}

	if v.unmarshaler == nil {
		v.unmarshaler = JsonValueUnmarshaler[T]()
	}

	parsed, err := v.unmarshaler(payload)
	if err != nil {
		v.log.With(log.Error(err)).Warn("Failed to unmarshal payload from mqtt")
		// TODO: Can/should we expose this error with a callback?
		return
	}

	v.log.With(slog.Any("v", parsed)).Debug("Received new value from mqtt")
	v.log.With(slog.Int("count", len(v.watchers))).Debug("Updating watchers")
	v.v, v.initialized = parsed, true
	for _, w := range v.watchers {
		// TODO: Call in separate goroutine? Do something like signal.Notify?
		w(v.v)
	}
}

// FullyQualifiedTopic calculates the MQTT Topic for this value when given the specified prefix. If the underlying
// RemoteValue (not the value it holds) is nil, the empty string is returned.
func (v *RemoteValue[T]) FullyQualifiedTopic(prefix string) string {
	if v == nil {
		return ""
	}

	return JoinTopic(prefix, v.topic)
}

// AppendSubscribeOptions adds a paho.SubscribeOptions value to the slice of existing options if this RemoteValue is not
// nil and has a configured topic.
func (v *RemoteValue[T]) AppendSubscribeOptions(existing []Subscription, prefix string) []Subscription {
	if v == nil || v.topic == "" {
		return existing
	}

	return append(existing, Subscription{
		Topic:   v.FullyQualifiedTopic(prefix),
		Options: v.opts,
	})
}

// Get returns the most recent value received from mqtt. If no value has been received yet, the second return value will
// be false.
func (v *RemoteValue[T]) Get() (T, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.v, v.initialized
}

// Watch registers a callback to execute when receiving new messages from mqtt. After receiving a new value from the
// router, it calls all watchers serially using the new value. Watchers should not block, any long operations executed
// in a watcher should start a new goroutine.
func (v *RemoteValue[T]) Watch(callback func(T)) int {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.log.Debug("Adding watcher")

	v.watchers = append(v.watchers, callback)
	return len(v.watchers) - 1
}

// Unwatch removes the specified callback from the watch list.
func (v *RemoteValue[T]) Unwatch(id int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.watchers == nil || id < 1 || id > len(v.watchers) {
		v.log.With(slog.Int("id", id), slog.Int("count", len(v.watchers))).Warn("Tried to remove an invalid watcher")
		return
	}

	v.log.With(slog.Int("id", id)).Debug("Removing watcher")

	v.watchers = append(v.watchers[:id], v.watchers[id+1:]...)
}

// DesiredValue makes calling RemoteValue.Await on comparable remote values easier
func DesiredValue[T comparable](v T) func(T) bool {
	return func(vv T) bool {
		return v == vv
	}
}

// Await watches for updates to this RemoteValue. When updated values pass the desired filter, the updated value is
// returned along with a nil error. Close the provided context to cancel. The watch is removed upon return.
//
// If the underlying type of this remote value is comparable, you can use DesiredValue to construct the check.
//
// Note that the underlying value of this RemoteValue may change between when the filter passes and when this function
// returns. The returned value is the first value to pass the desired filter function and may not be the underlying
// value for frequently updated values.
func (v *RemoteValue[T]) Await(ctx context.Context, desired func(T) bool) (T, error) {
	done := make(chan struct{})

	v.log.Debug("Awaiting value")

	var got T
	id := v.Watch(func(t T) {
		if desired(t) {
			v.log.Debug("Received expected value")

			got = t
			close(done)
		}
	})

	defer func() {
		v.Unwatch(id)
	}()

	select {
	case <-done:
		return got, nil
	case <-ctx.Done():
		v.log.Debug("Timeout waiting for value")
		return got, context.Cause(ctx)
	}
}
