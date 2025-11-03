package platform

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

// LightOnCommandType configures how Home Assistant sends style and power commands via MQTT for this component.
type LightOnCommandType string

const (
	// LightOnCommandTypeLast instructs Home Assistant to send any style (brightness, color, etc) topics first and then
	// a payload_on to the Light.Command. This is the default behavior.
	LightOnCommandTypeLast    LightOnCommandType = "last"
	DefaultLightOnCommandType                    = LightOnCommandTypeLast
	// LightOnCommandTypeFirst instructs Home Assistant to send the payload_on and then any style topics. Using
	LightOnCommandTypeFirst LightOnCommandType = "first"
	// LightOnCommandTypeBrightness instructs Home Assistant to only send brightness commands instead of the payload_on
	// to turn the light on.
	LightOnCommandTypeBrightness LightOnCommandType = "brightness"
)

// HueSat holds Hue and Saturation values for this component. It implements slog.LogValuer.
type HueSat struct {
	Hue        float64
	Saturation float64
}

func (h HueSat) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Float64("hue", h.Hue),
		slog.Float64("sat", h.Saturation),
	)
}

// RGB holds 8-bit Red, Green, and Blue values for a Light. It implements fmt.Stringer and slog.LogValuer.
type RGB struct {
	R, G, B uint8
}

func (r RGB) String() string {
	return fmt.Sprintf("#%02x%02x%02x", r.R, r.G, r.B)
}

func (r RGB) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Uint64("r", uint64(r.R)),
		slog.Uint64("g", uint64(r.G)),
		slog.Uint64("b", uint64(r.B)),
		slog.String("hex", r.String()),
	)
}

var (
	RGBMarshaler mqtt.ValueMarshaler[RGB] = func(v RGB) ([]byte, error) {
		return []byte(fmt.Sprintf("%d,%d,%d", v.R, v.G, v.B)), nil
	}
	RGBUnmarshaler mqtt.ValueUnmarshaler[RGB] = func(bytes []byte) (RGB, error) {
		parts := strings.Split(string(bytes), ",")
		if len(parts) != 3 {
			return RGB{}, fmt.Errorf("invalid RGB representation: %s", bytes)
		}

		r, errR := strconv.ParseUint(parts[0], 10, 8)
		g, errG := strconv.ParseUint(parts[1], 10, 8)
		b, errB := strconv.ParseUint(parts[2], 10, 8)

		return RGB{uint8(r), uint8(g), uint8(b)}, errors.Join(errR, errG, errB)
	}
)

// RGBW holds an 8-bit White value in addition to RGB values for a Light. It implements fmt.Stringer and slog.LogValuer.
type RGBW struct {
	RGB
	W uint8
}

func (r RGBW) String() string {
	return fmt.Sprintf("#%02x%02x%02x%02x", r.R, r.G, r.B, r.W)
}

func (r RGBW) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Uint64("r", uint64(r.R)),
		slog.Uint64("g", uint64(r.G)),
		slog.Uint64("b", uint64(r.B)),
		slog.Uint64("w", uint64(r.W)),
		slog.String("hex", r.String()),
	)
}

// RGBWW holds an additional 8-bit White value in addition to RGBW values for a Light. It implements fmt.Stringer and
// slog.LogValuer.
type RGBWW struct {
	RGBW
	WW uint8
}

func (r RGBWW) String() string {
	return fmt.Sprintf("#%02x%02x%02x%02x%02x", r.R, r.G, r.B, r.W, r.WW)
}

func (r RGBWW) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Uint64("r", uint64(r.R)),
		slog.Uint64("g", uint64(r.G)),
		slog.Uint64("b", uint64(r.B)),
		slog.Uint64("w", uint64(r.W)),
		slog.Uint64("ww", uint64(r.WW)),
		slog.String("hex", r.String()),
	)
}

type XY struct {
	X float64
	Y float64
}

func (xy XY) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Float64("x", xy.X),
		slog.Float64("y", xy.Y),
	)
}

// Light is a hqtt.Platform that implements the light.mqtt integration for Home Assistant.
//
// See https://www.home-assistant.io/integrations/light.mqtt/
type Light struct {
	// Defines when on the payload_on is sent.
	OnCommandType LightOnCommandType

	// Flag that defines if switch works in optimistic mode.
	Optimistic bool

	// The current state of the Light
	State *mqtt.Value[hass.PowerState]
	// Home Assistant will write commands for this entity to this value
	Command *mqtt.RemoteValue[hass.PowerState] `hqtt:"required"`

	// Custom values to use for payload commands
	CustomPowerStateValues hass.CustomPowerState

	// The Color Mode currently in use by the light. If this is not configured, it will be automatically set in Home
	// Assistant according to the last received valid color or color temperature. The unit used is mireds, or if
	// ColorTemperatureInKelvin is set to true, in Kelvin.
	ColorMode *mqtt.Value[hass.ColorMode]
	// Home Assistant will write the desired color mode to this value
	ColorModeCommand *mqtt.RemoteValue[hass.ColorMode]
	// The color modes supported by this light
	SupportedColorModes []hass.ColorMode

	// The current brightness of the light
	Brightness *mqtt.Value[uint]
	// Home Assistant will write desired brightness to this value
	BrightnessCommand *mqtt.RemoteValue[uint]
	// Defines the maximum brightness value (i.e., 100%). HomeAssistant will use 255 if not otherwise specified.
	BrightnessScale uint

	// The current color temperature of the light
	ColorTemperature *mqtt.Value[uint]
	// Home Assistant will write desired color temperature to this value
	ColorTemperatureCommand *mqtt.RemoteValue[uint]
	// Whether color temperature is in Kelvin (true) or mireds (false)
	ColorTemperatureInKelvin bool
	// The maximum color temperature in Kelvin. Defaults to 6535.
	MaxKelvin uint
	// The minimum color temperature in Kelvin. Defaults to 2000.
	MinKelvin uint
	// The maximum color temperature in mireds.
	MaxMireds uint
	// The minimum color temperature in mireds.
	MinMireds uint

	// The current Hue and Saturation values for this light
	HueSat *mqtt.Value[HueSat]
	// Home Assistant will write the desired Hue and Saturation values to this value
	HueSatCommand *mqtt.RemoteValue[HueSat]

	// The current XY values for this light
	XY *mqtt.Value[XY]
	// Home Assistant will write desired XY values to this value
	XYCommand *mqtt.RemoteValue[XY]

	// The current RGB Value for this light
	RGB *mqtt.Value[RGB]
	// Home Assistant will write desired RGB values to this value
	RGBCommand *mqtt.RemoteValue[RGB]

	// The current RGBW Value for this light
	RGBW *mqtt.Value[RGBW]
	// Home Assistant will write desired RGBW values to this value
	RGBWCommand *mqtt.RemoteValue[RGBW]

	// The current RGBWW Value for this light
	RGBWW *mqtt.Value[RGBWW]
	// Home Assistant will write desired RGBWW values to this value
	RGBWWCommand *mqtt.RemoteValue[RGBWW]

	// Home Assistant writes brightness values to this value when the light should operate in white mode.
	WhiteBrightnessCommand *mqtt.RemoteValue[uint]
	// The maximum white level (i.e., 100%) of this light. Defaults to 255 if not set.
	WhiteScale uint

	// The current effect the light is displaying
	Effect *mqtt.Value[string]
	// Home Assistant will write the desired effect to this value
	EffectCommand *mqtt.RemoteValue[string]
	// The list of possible effects this device supports
	PossibleEffects []string
}

func (l *Light) PlatformName() string {
	return "light"
}

func (l *Light) Subscriptions(prefix string) []mqtt.Subscription {
	var result []mqtt.Subscription

	result = l.Command.AppendSubscribeOptions(result, prefix)
	result = l.ColorModeCommand.AppendSubscribeOptions(result, prefix)
	result = l.BrightnessCommand.AppendSubscribeOptions(result, prefix)
	result = l.ColorTemperatureCommand.AppendSubscribeOptions(result, prefix)
	result = l.HueSatCommand.AppendSubscribeOptions(result, prefix)
	result = l.XYCommand.AppendSubscribeOptions(result, prefix)
	result = l.RGBCommand.AppendSubscribeOptions(result, prefix)
	result = l.RGBWCommand.AppendSubscribeOptions(result, prefix)
	result = l.RGBWWCommand.AppendSubscribeOptions(result, prefix)
	result = l.WhiteBrightnessCommand.AppendSubscribeOptions(result, prefix)
	result = l.EffectCommand.AppendSubscribeOptions(result, prefix)

	return result
}

// ServeMQTT handles the mqtt payload received on the specified topic suffix. It will route the payload to the first
// non-nil mqtt.RemoveValue that has a matching topic for the light. It is up to the user to ensure each configured
// mqtt.RemoteValue has a unique Topic configured.
func (l *Light) ServeMQTT(w mqtt.Writer, topic string, payload []byte) {
	switch topic {
	case l.Command.FullyQualifiedTopic(""):
		l.Command.ServeMQTT(w, topic, payload)
	case l.ColorModeCommand.FullyQualifiedTopic(""):
		l.ColorModeCommand.ServeMQTT(w, topic, payload)
	case l.BrightnessCommand.FullyQualifiedTopic(""):
		l.BrightnessCommand.ServeMQTT(w, topic, payload)
	case l.ColorTemperatureCommand.FullyQualifiedTopic(""):
		l.ColorTemperatureCommand.ServeMQTT(w, topic, payload)
	case l.HueSatCommand.FullyQualifiedTopic(""):
		l.HueSatCommand.ServeMQTT(w, topic, payload)
	case l.XYCommand.FullyQualifiedTopic(""):
		l.XYCommand.ServeMQTT(w, topic, payload)
	case l.RGBCommand.FullyQualifiedTopic(""):
		l.RGBCommand.ServeMQTT(w, topic, payload)
	case l.RGBWCommand.FullyQualifiedTopic(""):
		l.RGBWCommand.ServeMQTT(w, topic, payload)
	case l.RGBWWCommand.FullyQualifiedTopic(""):
		l.RGBWWCommand.ServeMQTT(w, topic, payload)
	case l.WhiteBrightnessCommand.FullyQualifiedTopic(""):
		l.WhiteBrightnessCommand.ServeMQTT(w, topic, payload)
	case l.EffectCommand.FullyQualifiedTopic(""):
		l.EffectCommand.ServeMQTT(w, topic, payload)
	default:
		// TODO: Log?
	}
}

func (l *Light) MarshalDiscoveryTo(e *jsontext.Encoder, prefix string) error {
	return errors.Join(
		discovery.MarshalStdIfNot(DefaultLightOnCommandType, e, discovery.FieldOnCommandType, l.OnCommandType),

		discovery.MaybeMarshalStdComparable(e, discovery.FieldOptimistic, l.Optimistic),

		discovery.MaybeMarshalValueTopic(e, discovery.FieldStateTopic, l.State, prefix),
		discovery.MarshalRequiredRemoteValueTopic("command", e, discovery.FieldCommandTopic, l.Command, prefix),

		discovery.MaybeMarshalStdComparable(e, discovery.FieldPayloadOn, l.CustomPowerStateValues.On),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldPayloadOff, l.CustomPowerStateValues.Off),

		discovery.MaybeMarshalStateAndCommandTopics(
			"color mode", e,
			discovery.FieldColorModeStateTopic, l.ColorMode,
			discovery.FieldColorModeCommandTopic, l.ColorModeCommand,
			prefix,
		),
		discovery.MaybeMarshalStd(e, discovery.FieldSupportedColorModes, &l.SupportedColorModes),
		discovery.MaybeMarshalStateAndCommandTopics(
			"brightness", e,
			discovery.FieldBrightnessStateTopic, l.Brightness,
			discovery.FieldBrightnessCommandTopic, l.BrightnessCommand,
			prefix,
		),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldBrightnessScale, l.BrightnessScale),

		discovery.MaybeMarshalStateAndCommandTopics(
			"color temperature", e,
			discovery.FieldColorTemperatureStateTopic, l.ColorTemperature,
			discovery.FieldColorTemperatureCommandTopic, l.ColorTemperatureCommand,
			prefix,
		),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldColorTemperatureInKelvin, l.ColorTemperatureInKelvin),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldMaxKelvin, l.MaxKelvin),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldMinKelvin, l.MinKelvin),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldMaxMireds, l.MaxMireds),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldMinMireds, l.MinMireds),

		discovery.MaybeMarshalStateAndCommandTopics(
			"hue sat", e,
			discovery.FieldHueSatStateTopic, l.ColorTemperature,
			discovery.FieldHueSatCommandTopic, l.ColorTemperatureCommand,
			prefix,
		),

		discovery.MaybeMarshalStateAndCommandTopics(
			"xy", e,
			discovery.FieldXYCommandTopic, l.XY,
			discovery.FieldXYStateTopic, l.XYCommand,
			prefix,
		),

		discovery.MaybeMarshalStateAndCommandTopics(
			"rgb", e,
			discovery.FieldRGBStateTopic, l.RGB,
			discovery.FieldRGBCommandTopic, l.RGBCommand,
			prefix,
		),
		discovery.MaybeMarshalStateAndCommandTopics(
			"rgbw", e,
			discovery.FieldRGBWStateTopic, l.RGBW,
			discovery.FieldRGBWCommandTopic, l.RGBWCommand,
			prefix,
		),
		discovery.MaybeMarshalStateAndCommandTopics(
			"rgbww", e,
			discovery.FieldRGBWWStateTopic, l.RGBWW,
			discovery.FieldRGBWWCommandTopic, l.RGBWWCommand,
			prefix,
		),

		discovery.MaybeMarshalRemoteValueTopic(e, discovery.FieldWhiteCommandTopic, l.WhiteBrightnessCommand, prefix),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldWhiteScale, l.WhiteScale),

		discovery.MaybeMarshalStateAndCommandTopics(
			"effect", e,
			discovery.FieldEffectStateTopic, l.Effect,
			discovery.FieldEffectCommandTopic, l.EffectCommand,
			prefix,
		),
		discovery.MaybeMarshalStdSlice(e, discovery.FieldEffectList, l.PossibleEffects),
	)
}
