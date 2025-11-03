package hqtt

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/url"
	"strings"

	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

// ErrComponentAlreadySubscribed is the error returned by Component.Subscribe when it has already been subscribed. Call
// Component.Unsubscribe first.
var ErrComponentAlreadySubscribed = errors.New("component already subscribed")

// Component exposes HomeAssistant components (sensors, switches, lights, etc.) associated with a given device. It
// implements json.MarshalerTo by encoding the component for a Home Assistant Device Discovery payload.
type Component[TPlatform Platform] struct {
	Platform    TPlatform
	TopicPrefix string

	// The name of the entity. Set to the empty string if only the device name is relevant.
	Name string

	// The category of the entity. See https://developers.home-assistant.io/docs/core/entity/#generic-properties
	EntityCategory string

	// The Icon to use in the frontend for this entity
	Icon string

	// Picture URL for the entity.
	Picture *url.URL

	// Identifies to home assistant whether this entity is available
	Availability *mqtt.Value[hass.Availability] `hqtt:"required"`
	// Custom values to use for available and unavailable states
	CustomAvailabilityValues hass.CustomAvailability

	// Use this value instead of name for automatic generation of the entity ID. For example, `light.foobar`. When used
	// without a UniqueID, the entity ID will update during restart or reload if the entity ID is available. If the
	// entity ID already exists, the entity ID will be created with a number at the end. When used with a UniqueID, the
	// DefaultEntityID is only used when the entity is added for the first time. When set, this overrides a
	// user-customized entity ID if the entity was deleted and added again.
	DefaultEntityID string

	// TODO: EnabledByDefault / DisabledByDefault?

	// An ID that uniquely identifies this light. If two lights have the same unique ID, Home Assistant will raise an
	// exception. Required when used with device-based discovery.
	UniqueID string `hqtt:"required"`

	// MQTT Options to use when publishing updates for this device
	WriteOptions mqtt.WriteOptions

	subscribedTopics []string
}

func (c *Component[TPlatform]) ForRemoval() RemoveComponent {
	return RemoveComponent{Platform: c.Platform.PlatformName()}
}

// Subscribe registers MQTT Subscriptions for fields in use by this Component using the provided
// mqtt.SubscriptionManager. The subscriptions can be removed by calling Unsubscribe.
//
// TODO: Wire LWT to availability.
func (c *Component[TPlatform]) Subscribe(ctx context.Context, s mqtt.Subscriber) error {
	if len(c.subscribedTopics) != 0 {
		return ErrComponentAlreadySubscribed
	}

	subscriptions := c.Platform.Subscriptions(c.TopicPrefix)
	c.subscribedTopics = make([]string, len(subscriptions))
	for i, subscription := range subscriptions {
		c.subscribedTopics[i] = subscription.Topic
	}

	return s.Subscribe(ctx, mqtt.HandlerFunc(func(w mqtt.Writer, topic string, payload []byte) {
		rest, ok := strings.CutPrefix(topic, mqtt.TrimTopic(c.TopicPrefix))
		if !ok {
			return
		}

		c.Platform.ServeMQTT(w, mqtt.TrimTopic(rest), payload)
	}), c.Platform.Subscriptions(c.TopicPrefix)...)
}

// Unsubscribe removes MQTT Subscriptions for fields in use by this Component from the provided
// mqtt.SubscriptionManager.
func (c *Component[TPlatform]) Unsubscribe(ctx context.Context, s mqtt.Subscriber) error {
	if len(c.subscribedTopics) == 0 {
		return nil
	}

	topics := c.subscribedTopics
	c.subscribedTopics = nil

	return s.Unsubscribe(ctx, topics...)
}

func (c *Component[TPlatform]) MarshalJSONTo(e *jsontext.Encoder) error {
	// TODO: Name: Home Assistant docs say "Can be set to `null` if only the device name is relevant." Does this mean
	//       omitted? The value should be a literal json null? The string "null"?
	nameToken := jsontext.Null
	if c.Name != "" {
		nameToken = jsontext.String(c.Name)
	}

	return errors.Join(
		e.WriteToken(jsontext.BeginObject),

		discovery.MarshalStdComparable("platform", e, discovery.FieldPlatform, c.Platform.PlatformName()),

		e.WriteToken(jsontext.String("name")),
		e.WriteToken(nameToken),

		discovery.MaybeMarshalStdComparable(e, discovery.FieldEntityCategory, c.EntityCategory),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldIcon, c.Icon),
		discovery.MaybeMarshalStd(e, discovery.FieldPicture, c.Picture),

		discovery.MarshalRequiredValueTopic("availability", e, discovery.FieldAvailabilityTopic, c.Availability, c.TopicPrefix),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldPayloadAvailable, c.CustomAvailabilityValues.Available),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldPayloadNotAvailable, c.CustomAvailabilityValues.Unavailable),

		discovery.MaybeMarshalStdComparable(e, discovery.FieldDefaultEntityID, c.DefaultEntityID),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldUniqueID, c.UniqueID),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldQualityOfService, c.WriteOptions.QoS),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldRetain, c.WriteOptions.Retain),

		c.Platform.MarshalDiscoveryTo(e, c.TopicPrefix),

		e.WriteToken(jsontext.EndObject),
	)
}

// RemoveComponent is used to remove a Component from device discovery. Construct a RemoveComponent with the appropriate
// platform name manually or use Component.ForRemoval.
type RemoveComponent struct {
	Platform string `json:"platform"`
}

func (r RemoveComponent) MarshalJSONTo(e *jsontext.Encoder) error {
	return json.MarshalEncode(e, &r)
}
