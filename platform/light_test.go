package platform

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

func setupTestLight(t *testing.T) (*Light, *jsontext.Encoder, *bytes.Buffer) {
	t.Helper()

	b := &bytes.Buffer{}

	return &Light{
		State:   mqtt.NewValue("state", hass.PowerStateMarshaler),
		Command: mqtt.NewRemoteValue("command", hass.PowerStateUnmarshaler),
	}, jsontext.NewEncoder(b), b
}

func TestLight_PlatformName(t *testing.T) {
	sut, _, _ := setupTestLight(t)

	require.Equal(t, "light", sut.PlatformName())
}

func TestLight_Subscriptions(t *testing.T) {
	sut, _, _ := setupTestLight(t)

	sut.ColorModeCommand = mqtt.NewRemoteValue("color_mode", hass.ColorModeUnmarshaler)
	sut.BrightnessCommand = mqtt.NewRemoteValue("brightness", mqtt.UintUnmarshaler)
	sut.ColorTemperatureCommand = mqtt.NewRemoteValue("color_temperature", mqtt.UintUnmarshaler)
	sut.HueSatCommand = mqtt.NewRemoteValue("hue_sat", HueSatUnmarshaler)
	sut.XYCommand = mqtt.NewRemoteValue("xy", XYUnmarshaler)
	sut.RGBCommand = mqtt.NewRemoteValue("rgb", RGBUnmarshaler)
	sut.RGBWCommand = mqtt.NewRemoteValue("rgbw", RGBWUnmarshaler)
	sut.RGBWWCommand = mqtt.NewRemoteValue("rgbww", RGBWWUnmarshaler)
	sut.WhiteBrightnessCommand = mqtt.NewRemoteValue("white_brightness", mqtt.UintUnmarshaler)
	sut.EffectCommand = mqtt.NewRemoteValue("effect", mqtt.StringUnmarshaler)

	result := sut.Subscriptions("sut")

	require.Len(t, result, 11)

	assertHasTopicSubscription(t, result, "sut/command")
	assertHasTopicSubscription(t, result, "sut/color_mode")
	assertHasTopicSubscription(t, result, "sut/brightness")
	assertHasTopicSubscription(t, result, "sut/color_temperature")
	assertHasTopicSubscription(t, result, "sut/hue_sat")
	assertHasTopicSubscription(t, result, "sut/xy")
	assertHasTopicSubscription(t, result, "sut/rgb")
	assertHasTopicSubscription(t, result, "sut/rgbw")
	assertHasTopicSubscription(t, result, "sut/rgbww")
	assertHasTopicSubscription(t, result, "sut/white_brightness")
	assertHasTopicSubscription(t, result, "sut/effect")
}

func TestLight_Handler(t *testing.T) {
	sut, _, _ := setupTestLight(t)

	sut.ColorModeCommand = mqtt.NewRemoteValue("color_mode", hass.ColorModeUnmarshaler)
	sut.BrightnessCommand = mqtt.NewRemoteValue("brightness", mqtt.UintUnmarshaler)
	sut.ColorTemperatureCommand = mqtt.NewRemoteValue("color_temperature", mqtt.UintUnmarshaler)
	sut.HueSatCommand = mqtt.NewRemoteValue("hue_sat", HueSatUnmarshaler)
	sut.XYCommand = mqtt.NewRemoteValue("xy", XYUnmarshaler)
	sut.RGBCommand = mqtt.NewRemoteValue("rgb", RGBUnmarshaler)
	sut.RGBWCommand = mqtt.NewRemoteValue("rgbw", RGBWUnmarshaler)
	sut.RGBWWCommand = mqtt.NewRemoteValue("rgbww", RGBWWUnmarshaler)
	sut.WhiteBrightnessCommand = mqtt.NewRemoteValue("white_brightness", mqtt.UintUnmarshaler)
	sut.EffectCommand = mqtt.NewRemoteValue("effect", mqtt.StringUnmarshaler)

	sut.ServeMQTT(nil, "command", []byte(hass.PowerStateOn))
	sut.ServeMQTT(nil, "color_mode", []byte(hass.ColorModeBrightness))
	sut.ServeMQTT(nil, "brightness", []byte("123"))
	sut.ServeMQTT(nil, "color_temperature", []byte("456"))
	sut.ServeMQTT(nil, "hue_sat", []byte(`789,987`))
	sut.ServeMQTT(nil, "xy", []byte(`0.123,0.456`))
	sut.ServeMQTT(nil, "rgb", []byte(`111,112,113`))
	sut.ServeMQTT(nil, "rgbw", []byte(`121,122,123,124`))
	sut.ServeMQTT(nil, "rgbww", []byte(`131,132,133,134,135`))
	sut.ServeMQTT(nil, "white_brightness", []byte("999"))
	sut.ServeMQTT(nil, "effect", []byte("rainbow"))

	assertHasRemoteValue(t, sut.Command, hass.PowerStateOn)
	assertHasRemoteValue(t, sut.ColorModeCommand, hass.ColorModeBrightness)
	assertHasRemoteValue(t, sut.BrightnessCommand, 123)
	assertHasRemoteValue(t, sut.ColorTemperatureCommand, 456)
	assertHasRemoteValue(t, sut.HueSatCommand, HueSat{Hue: 789, Saturation: 987})
	assertHasRemoteValue(t, sut.XYCommand, XY{X: 0.123, Y: 0.456})
	assertHasRemoteValue(t, sut.RGBCommand, RGB{R: 111, G: 112, B: 113})
	assertHasRemoteValue(t, sut.RGBWCommand, RGBW{R: 121, G: 122, B: 123, W: 124})
	assertHasRemoteValue(t, sut.RGBWWCommand, RGBWW{R: 131, G: 132, B: 133, W: 134, WW: 135})
	assertHasRemoteValue(t, sut.WhiteBrightnessCommand, 999)
	assertHasRemoteValue(t, sut.EffectCommand, "rainbow")
}

func TestLight_Discovery(t *testing.T) {
	t.Parallel()

	t.Run("Required", testRequiredDiscoveryFields(
		t,
		setupTestLight,
		regexpStateTopic,
		`"cmd_t"\n"sut/command"`,
	))

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		t.Run("LightOnCommandType", testOptional(t, setupTestLight, func(l *Light) {
			l.OnCommandType = LightOnCommandTypeBrightness
		}, `"on_cmd_type"\n"brightness"`, `"on_cmd_type"`))

		t.Run("Optimistic", testOptional(t, setupTestLight, func(l *Light) {
			l.Optimistic = true
		}, `"opt"\ntrue`, `"opt"`))

		t.Run("PayloadOn", testOptional(t, setupTestLight, func(l *Light) {
			l.CustomPowerStateValues = hass.CustomPowerState{On: "foo"}
		}, `"pl_on"\n"foo"`, `"pl_on"`))

		t.Run("PayloadOff", testOptional(t, setupTestLight, func(l *Light) {
			l.CustomPowerStateValues = hass.CustomPowerState{Off: "bar"}
		}, `"pl_off"\n"bar"`, `"pl_off"`))

		t.Run("ColorMode", testOptional(t, setupTestLight, func(l *Light) {
			l.ColorMode = mqtt.NewValue("color_mode", hass.ColorModeMarshaler)
			l.ColorModeCommand = mqtt.NewRemoteValue("color_mode/set", hass.ColorModeUnmarshaler)
		}, `"clrm_stat_t"\n"sut/color_mode"\n"clrm_cmd_t"\n"sut/color_mode/set"`, `"clrm_stat_t"\n.*"clrm_cmd_t"`))

		t.Run("SupportedColorModes", testOptional(t, setupTestLight, func(l *Light) {
			l.SupportedColorModes = []hass.ColorMode{hass.ColorModeBrightness, hass.ColorModeHueSat}
		}, `"sup_clrm"\n\["brightness",\s*"hs"\]`, `"sup_clrm"`))

		t.Run("Brightness", testOptional(t, setupTestLight, func(l *Light) {
			l.Brightness = mqtt.NewValue("brightness", mqtt.UintMarshaler)
			l.BrightnessCommand = mqtt.NewRemoteValue("brightness/set", mqtt.UintUnmarshaler)

		}, `"bri_stat_t"\n"sut/brightness"\n"bri_cmd_t"\n"sut/brightness/set"`, `"bri_stat_t"\n.*"bri_cmd_t"`))

		t.Run("BrightnessScale", testOptional(t, setupTestLight, func(l *Light) {
			l.BrightnessScale = 123
		}, `"bri_scl"\n123`, `"bri_scl"`))

		t.Run("ColorTemperature", testOptional(t, setupTestLight, func(l *Light) {
			l.ColorTemperature = mqtt.NewValue("color_temp", mqtt.UintMarshaler)
			l.ColorTemperatureCommand = mqtt.NewRemoteValue("color_temp/set", mqtt.UintUnmarshaler)
		}, `"clr_temp_stat_t"\n"sut/color_temp"\n"clr_temp_cmd_t"\n"sut/color_temp/set"`, `"clr_temp_stat_t"\n.*"clr_temp_cmd_t"`))

		t.Run("ColorTemperatureInKelvin", testOptional(t, setupTestLight, func(l *Light) {
			l.ColorTemperatureInKelvin = true
		}, `"clr_temp_k"\ntrue`, `"clr_temp_k"`))

		t.Run("MaxKelvin", testOptional(t, setupTestLight, func(l *Light) {
			l.MaxKelvin = 123
		}, `"max_k"\n123`, `"max_k"`))
		t.Run("MinKelvin", testOptional(t, setupTestLight, func(l *Light) {
			l.MinKelvin = 456
		}, `"min_k"\n456`, `"min_k"`))

		t.Run("MaxMireds", testOptional(t, setupTestLight, func(l *Light) {
			l.MaxMireds = 987
		}, `"max_mirs"\n987`, `"max_mirs"`))
		t.Run("MaxMireds", testOptional(t, setupTestLight, func(l *Light) {
			l.MinMireds = 654
		}, `"min_mirs"\n654`, `"min_mirs"`))

		t.Run("HueSat", testOptional(t, setupTestLight, func(l *Light) {
			l.HueSat = mqtt.NewValue("hue_sat", HueSatMarshaler)
			l.HueSatCommand = mqtt.NewRemoteValue("hue_sat/set", HueSatUnmarshaler)
		}, `"hs_stat_t"\n"sut/hue_sat"\n"hs_cmd_t"\n"sut/hue_sat/set"`, `"hs_stat_t"\n.*"hs_cmd_t"`))

		t.Run("XY", testOptional(t, setupTestLight, func(l *Light) {
			l.XY = mqtt.NewValue("xy", XYMarshaler)
			l.XYCommand = mqtt.NewRemoteValue("xy/set", XYUnmarshaler)
		}, `"xy_stat_t"\n"sut/xy"\n"xy_cmd_t"\n"sut/xy/set"`, `"xy_stat_t"\n.*"xy_cmd_t"`))

		t.Run("RGB", testOptional(t, setupTestLight, func(l *Light) {
			l.RGB = mqtt.NewValue("rgb", RGBMarshaler)
			l.RGBCommand = mqtt.NewRemoteValue("rgb/set", RGBUnmarshaler)
		}, `"rgb_stat_t"\n"sut/rgb"\n"rgb_cmd_t"\n"sut/rgb/set"`, `"rgb_stat_t"\n.*"rgb_cmd_t"`))

		t.Run("RGBW", testOptional(t, setupTestLight, func(l *Light) {
			l.RGBW = mqtt.NewValue("rgbw", RGBWMarshaler)
			l.RGBWCommand = mqtt.NewRemoteValue("rgbw/set", RGBWUnmarshaler)
		}, `"rgbw_stat_t"\n"sut/rgbw"\n"rgbw_cmd_t"\n"sut/rgbw/set"`, `"rgbw_stat_t"\n.*"rgbw_cmd_t"`))

		t.Run("RGBWW", testOptional(t, setupTestLight, func(l *Light) {
			l.RGBWW = mqtt.NewValue("rgbww", RGBWWMarshaler)
			l.RGBWWCommand = mqtt.NewRemoteValue("rgbww/set", RGBWWUnmarshaler)
		}, `"rgbww_stat_t"\n"sut/rgbww"\n"rgbww_cmd_t"\n"sut/rgbww/set"`, `"rgbww_stat_t"\n.*"rgbww_cmd_t"`))

		t.Run("WhiteBrightnessCommand", testOptional(t, setupTestLight, func(l *Light) {
			l.WhiteBrightnessCommand = mqtt.NewRemoteValue("white_brightness/set", mqtt.UintUnmarshaler)
		}, `"whit_cmd_t"\n"sut/white_brightness/set"`, `"whit_cmd_t"`))

		t.Run("WhiteScale", testOptional(t, setupTestLight, func(l *Light) {
			l.WhiteScale = 123
		}, `"whit_scl"\n123`, `"whit_scl"`))

		t.Run("Effect", testOptional(t, setupTestLight, func(l *Light) {
			l.Effect = mqtt.NewValue("effect", mqtt.StringMarshaler)
			l.EffectCommand = mqtt.NewRemoteValue("effect/set", mqtt.StringUnmarshaler)
		}, `"fx_stat_t"\n"sut/effect"\n"fx_cmd_t"\n"sut/effect/set"`, `"fx_stat_t"\n.*"fx_cmd_t"`))

		t.Run("EffectList", testOptional(t, setupTestLight, func(l *Light) {
			l.PossibleEffects = []string{"A", "B", "C"}
		}, `"fx_list"\n\["A","B","C"\]`, `"fx_list"`))
	})
}
