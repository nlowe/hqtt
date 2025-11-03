package platform

import (
	"encoding/json/jsontext"
	"errors"
	"time"

	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

// Sensor is a hqtt.Platform that implements the sensor.mqtt integration for Home Assistant. The state of this sensor
// has a type of TValue, and attributes for state have a type of TAttributes.
//
// See the Home Assistant documentation for more details: https://www.home-assistant.io/integrations/sensor.mqtt/.
type Sensor[TValue, TAttributes any] struct {
	// If set, it defines the number of seconds after the sensor’s state expires if it’s not updated. After expiry, the
	// sensor’s state becomes unavailable. By default, the sensor’s state never expires. Note that when a sensor’s value
	// was sent retained to the MQTT broker, the last value sent will be replayed by the MQTT broker when Home Assistant
	// restarts or is reloaded. As this could cause the sensor to become available with an expired state, it is not
	// recommended to retain the sensor’s state payload at the MQTT broker. Home Assistant will store and restore the
	// sensor’s state for you and calculate the remaining time to retain the sensor’s state before it becomes
	// unavailable.
	ExpireMeasurementsAfter time.Duration

	// Instruct Home Assistant to calculate update events even if the value hasn’t changed. Useful if you want to have
	// meaningful value graphs in history.
	ForceUpdate bool

	// Attributes exposes state attributes for this sensor. Writes to this value imply ForceUpdate of the current sensor
	// state when a message is received on this topic by Home Assistant. For standard marshaling, use
	// mqtt.JsonValueMarshaler for the mqtt.ValueMarshaler for this value. When using a custom marshaler, the resulting
	// byte slice must be a json string.
	Attributes *mqtt.Value[TAttributes]

	// List of allowed sensor state value. The sensor’s device_class must be set to enum. The options option cannot be
	// used together with state_class or unit_of_measurement.
	//
	// Note: The Home Assistant documentation states "an empty list is not allowed". Empty / nil slices will be omitted
	// when marshaling discovery information.
	EnumOptions []TValue

	// The number of decimals which should be used in the sensor’s state after rounding.
	SuggestedDisplayPrecision uint

	// The hass.StateClass of the sensor.
	StateClass hass.StateClass

	// The current value of the sensor
	State *mqtt.Value[TValue] `hqtt:"required"`

	// Defines the units used by this sensor
	// TODO: Can/should we type this and grab constants from Home Assistant?
	UnitOfMeasurement string
}

func (s *Sensor[TValue, TAttributes]) PlatformName() string {
	return "sensor"
}

func (s *Sensor[TValue, TAttributes]) Subscriptions(_ string) []mqtt.Subscription {
	return nil
}

func (s *Sensor[TValue, TAttributes]) ServeMQTT(_ mqtt.Writer, _ string, _ []byte) {}

func (s *Sensor[TValue, TAttributes]) MarshalDiscoveryTo(e *jsontext.Encoder, prefix string) error {
	return errors.Join(
		discovery.MaybeMarshalStdComparable(e, discovery.FieldExpireMeasurementsAfter, s.ExpireMeasurementsAfter),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldForceUpdate, s.ForceUpdate),
		discovery.MaybeMarshalValueTopic(e, discovery.FieldAttributesTopic, s.Attributes, prefix),
		discovery.MaybeMarshalStdSlice(e, discovery.FieldOptions, s.EnumOptions),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldSuggestedDisplayPrecision, s.SuggestedDisplayPrecision),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldStateClass, s.StateClass),
		discovery.MarshalRequiredValueTopic("state", e, discovery.FieldStateTopic, s.State, prefix),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldUnitOfMeasurement, s.UnitOfMeasurement),
	)
}

// NewSensorAttributeValue constructs a mqtt.Value for the provided attribute type. If marshaler is nil, it uses
// mqtt.JsonValueMarshaler to marshal values.
func NewSensorAttributeValue[TAttributes any](topic string, marshaler mqtt.ValueMarshaler[TAttributes]) *mqtt.Value[TAttributes] {
	if marshaler == nil {
		marshaler = mqtt.JsonValueMarshaler[TAttributes]()
	}

	return mqtt.NewValue[TAttributes](topic, marshaler)
}
