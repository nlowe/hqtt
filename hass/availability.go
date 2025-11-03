package hass

import (
	"log/slog"

	"github.com/nlowe/hqtt/mqtt"
)

// Availability exposes whether Home Assistant should consider a device or entity as "available" (aka it is online).
type Availability string

var (
	AvailabilityMarshaler mqtt.ValueMarshaler[Availability] = func(v Availability) ([]byte, error) {
		return mqtt.StringMarshaler(string(v))
	}
	AvailabilityUnmarshaler mqtt.ValueUnmarshaler[Availability] = func(bytes []byte) (Availability, error) {
		v, err := mqtt.StringUnmarshaler(bytes)
		return Availability(v), err
	}
)

const (
	// Available is the Availability value for online/available devices.
	Available Availability = "online"
	// Unavailable is the Availability value for offline/unavailable devices.
	Unavailable Availability = "offline"
)

// CustomAvailability instructs Home Assistant to use different values to determine availability state. It implements
// slog.LogValuer.
type CustomAvailability struct {
	Available   Availability
	Unavailable Availability
}

func (c CustomAvailability) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("available_value", string(c.Available)),
		slog.String("unavailable_value", string(c.Unavailable)),
	)
}
