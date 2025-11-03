package discovery

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nlowe/hqtt/mqtt"
)

var (
	// ErrValueRequired is the error returned by marshal functions for values that hold the type's associated Zero value
	// when marshaling the discovery payload.
	ErrValueRequired = errors.New("value is required")
	// ErrTopicRequired is the error returned by MarshalRequiredTopic, MarshalRequiredValueTopic, and
	// MarshalRequiredRemoteValueTopic when the provided topic is empty (usually because the required value is nil).
	ErrTopicRequired = errors.New("topic is required")
	// ErrMissingStateOrCommandTopic is the error returned by MaybeMarshalStateAndCommandTopics if either the state
	// topic or the command topic (but not both) are specified.
	ErrMissingStateOrCommandTopic = errors.New("state and command topics must both be configured")

	// Marshalers contains json.Marshalers for types from the standard library to make them conform to the Home
	// Assistant MQTT Device Discovery schema (e.g. render URLs as strings).
	Marshalers = json.JoinMarshalers(
		// Marshal URLs as their string representation
		json.MarshalToFunc[*url.URL](func(e *jsontext.Encoder, u *url.URL) error {
			return e.WriteToken(jsontext.String(u.String()))
		}),
		// Marshal durations as integer seconds
		json.MarshalToFunc[time.Duration](func(e *jsontext.Encoder, t time.Duration) error {
			return e.WriteToken(jsontext.Int(int64(t.Seconds())))
		}),
	)
)

// MarshalRequiredTopic encodes the topic for the discovery payload being built. It returns ErrTopicRequired if the
// topic is the empty string.
func MarshalRequiredTopic(name string, e *jsontext.Encoder, k string, topic string) error {
	if topic == "" {
		return fmt.Errorf("%s: %w", name, ErrTopicRequired)
	}

	return MaybeMarshalTopic(e, k, topic)
}

// MarshalRequiredValueTopic encodes the topic for the provided mqtt.Value. It returns ErrTopicRequired if the value is
// nil or has no configured topic.
func MarshalRequiredValueTopic[T any](name string, e *jsontext.Encoder, k string, v *mqtt.Value[T], prefix string) error {
	return MarshalRequiredTopic(name, e, k, v.FullyQualifiedTopic(prefix))
}

// MarshalRequiredRemoteValueTopic encodes the topic for the provided mqtt.RemoteValue. It returns ErrTopicRequired if
// the value is nil or has no configured topic.
func MarshalRequiredRemoteValueTopic[T any](name string, e *jsontext.Encoder, k string, v *mqtt.RemoteValue[T], prefix string) error {
	return MarshalRequiredTopic(name, e, k, v.FullyQualifiedTopic(prefix))
}

// MaybeMarshalTopic encodes the topic for the discovery payload being built if the topic string is not empty.
func MaybeMarshalTopic(e *jsontext.Encoder, k string, topic string) error {
	if topic == "" {
		return nil
	}

	return errors.Join(
		e.WriteToken(jsontext.String(k)),
		e.WriteToken(jsontext.String(topic)),
	)
}

// MaybeMarshalValueTopic encodes the topic for the provided mqtt.Value if the topic string is not empty.
func MaybeMarshalValueTopic[T any](e *jsontext.Encoder, k string, v *mqtt.Value[T], prefix string) error {
	return MaybeMarshalTopic(e, k, v.FullyQualifiedTopic(prefix))
}

// MaybeMarshalRemoteValueTopic encodes the topic for the provided mqtt.RemoteValue if the topic string is not empty.
func MaybeMarshalRemoteValueTopic[T any](e *jsontext.Encoder, k string, v *mqtt.RemoteValue[T], prefix string) error {
	return MaybeMarshalTopic(e, k, v.FullyQualifiedTopic(prefix))
}

// MaybeMarshalStateAndCommandTopics marshals the specified state (mqtt.Value) and command (mqtt.RemoteValue) topics if
// they are not nil. If one is not nil, the other must also not be nil.
//
// Note: Go cannot currently infer the type parameter of T for calls in the form `Foo[T any, TValue T1[T] | T2[T]]` if
// T1 and T2 have different shapes (like mqtt.Value and mqtt.RemoteValue do). When calling this you will typically have
// to provide at least the type parameter T, which will be the type that the underlying value holds.
func MaybeMarshalStateAndCommandTopics[T any](name string, e *jsontext.Encoder, sk string, s *mqtt.Value[T], ck string, c *mqtt.RemoteValue[T], prefix string) error {
	if s == nil && c == nil {
		return nil
	}

	if s == nil || c == nil {
		return fmt.Errorf("%s: %w", name, ErrMissingStateOrCommandTopic)
	}

	return errors.Join(
		MarshalRequiredValueTopic(name, e, sk, s, prefix),
		MarshalRequiredRemoteValueTopic(name, e, ck, c, prefix),
	)
}

// MarshalStd marshals the specified value using json.MarshalEncode with Marshalers. If the provided value is nil, it
// returns ErrValueRequired.
func MarshalStd[T any](name string, e *jsontext.Encoder, k string, v *T) error {
	if v == nil {
		return fmt.Errorf("%s: %w", name, ErrValueRequired)
	}

	return MaybeMarshalStd(e, k, v)
}

// MaybeMarshalStd marshals the provided value using json.MarshalEncode with Marshalers if it is not nil.
func MaybeMarshalStd[T any](e *jsontext.Encoder, k string, v *T) error {
	if v == nil {
		return nil
	}

	return errors.Join(
		e.WriteToken(jsontext.String(k)),
		json.MarshalEncode(e, v, json.WithMarshalers(Marshalers)),
	)
}

// MaybeMarshalStdSlice marshals the provided slice of values using json.MarshalEncode with Marshalers if it is not
// empty.
func MaybeMarshalStdSlice[T any](e *jsontext.Encoder, k string, v []T) error {
	if len(v) == 0 {
		return nil
	}

	return errors.Join(
		e.WriteToken(jsontext.String(k)),
		json.MarshalEncode(e, v, json.WithMarshalers(Marshalers)),
	)
}

// MarshalStdComparable marshals the provided value using Marshalers. If it is equal to the type's zero value, it
// returns ErrValueRequired.
func MarshalStdComparable[T comparable](name string, e *jsontext.Encoder, k string, v T) error {
	var defaultT T
	if v == defaultT {
		return fmt.Errorf("%s: %w", name, ErrValueRequired)
	}

	return MaybeMarshalStd(e, k, &v)
}

// MaybeMarshalStdComparable marshals the provided value using Marshalers if it is not equal to the type's zero value.
func MaybeMarshalStdComparable[T comparable](e *jsontext.Encoder, k string, v T) error {
	var defaultT T
	if v == defaultT {
		return nil
	}

	return MaybeMarshalStd(e, k, &v)
}

// MarshalStdIfNot marshals the provided value using Marshalers if it is not equal to the specified value.
func MarshalStdIfNot[T comparable](not T, e *jsontext.Encoder, vk string, v T) error {
	var defaultT T
	if v == not || v == defaultT {
		return nil
	}

	return errors.Join(
		e.WriteToken(jsontext.String(vk)),
		json.MarshalEncode(e, v, json.WithMarshalers(Marshalers)),
	)
}

// MaybeInlineMarshalStd marshals the provided map of values inline (without emitting jsontext.BeginObject and
// jsontext.EndObject tokens) using map keys for string tokens and json.MarshalEncode with Marshalers to marshal the
// values.
func MaybeInlineMarshalStd[T any, TMap map[string]T](e *jsontext.Encoder, v TMap) error {
	if len(v) == 0 {
		return nil
	}

	var err error
	for vk, vv := range v {
		err = errors.Join(
			err,
			e.WriteToken(jsontext.String(vk)),
			json.MarshalEncode(e, vv, json.WithMarshalers(Marshalers)),
		)
	}

	return err
}
