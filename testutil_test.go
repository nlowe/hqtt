package hqtt

import (
	"encoding/json/jsontext"
	"errors"

	"github.com/nlowe/hqtt/mqtt"
	"github.com/nlowe/hqtt/mqtt/adapter/testqtt"
)

type testPlatform struct {
	testqtt.Recorder

	MarshalError       error
	SubscriptionConfig []mqtt.Subscription
}

var _ Platform = &testPlatform{}

func (t *testPlatform) ServeMQTT(_ mqtt.Writer, topic string, message []byte) {
	t.Record(testqtt.Message{
		Topic: topic,
		Data:  message,
	})
}

func (t *testPlatform) MarshalPlatformDiscoveryTo(e *jsontext.Encoder, prefix string) error {
	if t.MarshalError != nil {
		return t.MarshalError
	}

	return errors.Join(
		e.WriteToken(jsontext.String("platform_topic_prefix")),
		e.WriteToken(jsontext.String(prefix)),
	)
}

func (t *testPlatform) PlatformName() string {
	return "test"
}

func (t *testPlatform) Subscriptions(prefix string) []mqtt.Subscription {
	result := make([]mqtt.Subscription, len(t.SubscriptionConfig))
	for i, s := range t.SubscriptionConfig {
		result[i] = mqtt.Subscription{
			Topic:   mqtt.JoinTopic(prefix, s.Topic),
			Options: s.Options,
		}
	}

	return result
}
