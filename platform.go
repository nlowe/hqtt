package hqtt

import (
	"encoding/json/jsontext"

	"github.com/nlowe/hqtt/mqtt"
)

// Platform is the interface implemented by every MQTT Entity Component type.
type Platform interface {
	mqtt.Handler

	// MarshalDiscoveryTo marshals MQTT Device Discovery information to the specified jsontext.Encoder using the
	// provided prefix for all MQTT Topics.
	MarshalDiscoveryTo(e *jsontext.Encoder, prefix string) error

	// PlatformName returns the value for the `platform` field when configuring a component using this platform for MQTT
	// Device Discovery.
	PlatformName() string

	// Subscriptions returns the set of paho.SubscribeOptions for configured fields of this component. Only fields that
	// are properly configured should be included. Typically, each subscription is individually subscribed to, but other
	// mqtt.Subscriber implementations may choose to group topics with wildcards.
	Subscriptions(prefix string) []mqtt.Subscription
}
