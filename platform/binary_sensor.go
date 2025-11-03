package platform

import (
	"encoding/json/jsontext"
	"errors"
	"time"

	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

// BinarySensor is a Sensor that uses hass.PowerState for its state type (i.e. hass.PowerStateOn or hass.PowerStateOff).
//
// See Sensor for details about state attributes, and https://www.home-assistant.io/integrations/binary_sensor.mqtt/ for
// complete documentation.
type BinarySensor[TAttributes any] struct {
	Sensor[hass.PowerState, TAttributes]

	// For sensors that only send on state updates (like PIRs), this variable sets a delay in seconds after which the
	// sensor’s state will be updated back to off by Home Assistant.
	OffDelay time.Duration
}

func (s *BinarySensor[TAttributes]) PlatformName() string {
	return "binary_sensor"
}

func (s *BinarySensor[TAttributes]) Subscriptions(prefix string) []mqtt.Subscription {
	return s.Sensor.Subscriptions(prefix)
}

func (s *BinarySensor[TAttributes]) ServeMQTT(w mqtt.Writer, topic string, message []byte) {
	s.Sensor.ServeMQTT(w, topic, message)
}

func NewBinarySensor[TAttributes any](state *mqtt.Value[hass.PowerState], attrs *mqtt.Value[TAttributes]) *BinarySensor[TAttributes] {
	return &BinarySensor[TAttributes]{
		Sensor: Sensor[hass.PowerState, TAttributes]{
			State:      state,
			Attributes: attrs,
		},
	}
}

func (s *BinarySensor[TAttributes]) MarshalDiscoveryTo(e *jsontext.Encoder, prefix string) error {
	return errors.Join(
		s.Sensor.MarshalDiscoveryTo(e, prefix),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldOffDelay, s.OffDelay),
	)
}
