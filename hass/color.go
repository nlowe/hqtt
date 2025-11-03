package hass

import "github.com/nlowe/hqtt/mqtt"

// ColorMode represents constants that Home Assistant uses to determine what mode a given color represents
type ColorMode string

var (
	ColorModeMarshaler mqtt.ValueMarshaler[ColorMode] = func(v ColorMode) ([]byte, error) {
		return mqtt.StringMarshaler(string(v))
	}
	ColorModeUnmarshaler mqtt.ValueUnmarshaler[ColorMode] = func(bytes []byte) (ColorMode, error) {
		v, err := mqtt.StringUnmarshaler(bytes)
		return ColorMode(v), err
	}
)

const (
	ColorModeOnOff       ColorMode = "onoff"
	ColorModeBrightness  ColorMode = "brightness"
	ColorModeTemperature ColorMode = "color_temp"
	ColorModeHueSat      ColorMode = "hs"
	ColorModeXY          ColorMode = "xy"
	ColorModeRGB         ColorMode = "rgb"
	ColorModeRGBW        ColorMode = "rgbw"
	ColorModeRGBWW       ColorMode = "rgbww"
	ColorModeWhite       ColorMode = "white"
)
