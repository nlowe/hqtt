package platform

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt"
	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

func setupTestUpdate(t *testing.T) (*Update, *jsontext.Encoder, *bytes.Buffer) {
	t.Helper()

	b := &bytes.Buffer{}

	return &Update{
		State: mqtt.NewValue("state", mqtt.JsonValueMarshaler[UpdateState]()),
	}, jsontext.NewEncoder(b), b
}

func TestUpdate_PlatformName(t *testing.T) {
	sut, _, _ := setupTestUpdate(t)

	require.Equal(t, "update", sut.PlatformName())
}

func TestUpdate_Subscriptions(t *testing.T) {
	sut, _, _ := setupTestUpdate(t)

	sut.Command = mqtt.NewRemoteValue("command", mqtt.StringUnmarshaler)

	result := sut.Subscriptions("sut")

	require.Len(t, result, 1)

	assertHasTopicSubscription(t, result, "sut/command")
}

func TestUpdate_Handler(t *testing.T) {
	sut, _, _ := setupTestUpdate(t)

	sut.Command = mqtt.NewRemoteValue("command", mqtt.StringUnmarshaler)

	sut.ServeMQTT(nil, "command", []byte("install"))

	assertHasRemoteValue(t, sut.Command, "install")
}

func TestUpdate_Discovery(t *testing.T) {
	t.Parallel()

	t.Run("Required", testRequiredDiscoveryFields(
		t,
		setupTestUpdate,
		regexpStateTopic,
	))

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		t.Run("InstallPayload", testOptional(t, setupTestUpdate, func(u *Update) {
			u.InstallPayload = "install"
		}, `"pl_inst"\n"install"`, `"pl_inst"`))

		t.Run("Command", testOptional(t, setupTestUpdate, func(u *Update) {
			u.Command = mqtt.NewRemoteValue("command", mqtt.StringUnmarshaler)
		}, `"cmd_t"\n"sut/command"`, `"cmd_t"`))

		t.Run("Device", func(t *testing.T) {
			t.Parallel()

			t.Run("Valid", testOptional(t, setupTestUpdate, func(u *Update) {
				u.Device = &hqtt.Device{
					Name:        "Test Device",
					Identifiers: []string{"test"},
				}
			}, `"dev"\n{"name":"Test Device","ids":\["test"\]}`, `"dev"`))

			t.Run("Invalid", func(t *testing.T) {
				sut, e, _ := setupTestUpdate(t)

				sut.Device = &hqtt.Device{Name: "Test Device"}

				require.ErrorIs(t, sut.MarshalPlatformDiscoveryTo(e, "sut"), hqtt.ErrInvalidDevice)
			})
		})

		t.Run("DeviceClass", testOptional(t, setupTestUpdate, func(u *Update) {
			u.DeviceClass = hass.UpdateDeviceClassFirmware
		}, `"dev_cla"\n"firmware"`, `"dev_cla"`))

		t.Run("DisplayPrecision", testOptional(t, setupTestUpdate, func(u *Update) {
			u.DisplayPrecision = 3
		}, `"dsp_prc"\n3`, `"dsp_prc"`))
	})
}
