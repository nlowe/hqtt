package hass

import (
	"log/slog"

	"github.com/nlowe/hqtt/mqtt"
)

// PowerState represents generic on/off state for devices. This may or may not refer to physical power depending on the
// underlying entity (For example, a motion sensor may return PowerStateOn when motion is detected).
type PowerState string

var (
	PowerStateMarshaler mqtt.ValueMarshaler[PowerState] = func(v PowerState) ([]byte, error) {
		return mqtt.StringMarshaler(string(v))
	}

	PowerStateUnmarshaler mqtt.ValueUnmarshaler[PowerState] = func(bytes []byte) (PowerState, error) {
		v, err := mqtt.StringUnmarshaler(bytes)
		return PowerState(v), err
	}
)

const (
	PowerStateOn      PowerState = "ON"
	PowerStateOff     PowerState = "OFF"
	PowerStateUnknown PowerState = "None"
)

// CustomPowerState provides a way to configure custom values for on and off states for a given entity. It implements
// slog.LogValuer.
type CustomPowerState struct {
	On  PowerState
	Off PowerState
}

func (c CustomPowerState) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("on_value", string(c.On)),
		slog.String("off_value", string(c.Off)),
	)
}
