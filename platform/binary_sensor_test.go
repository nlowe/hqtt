package platform

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

func setupTestBinarySensor(t *testing.T) (*BinarySensor[map[string]any], *jsontext.Encoder, *bytes.Buffer) {
	t.Helper()

	b := &bytes.Buffer{}

	return &BinarySensor[map[string]any]{
		State: mqtt.NewValue("state", hass.PowerStateMarshaler),
	}, jsontext.NewEncoder(b), b
}

func TestBinarySensor_PlatformName(t *testing.T) {
	sut, _, _ := setupTestBinarySensor(t)

	require.Equal(t, "binary_sensor", sut.PlatformName())
}

func TestBinarySensor_NoSubscriptions(t *testing.T) {
	sut, _, _ := setupTestBinarySensor(t)

	require.Empty(t, sut.Subscriptions(""))
}

func TestBinarySensor_Discovery(t *testing.T) {
	t.Parallel()

	t.Run("Required", testRequiredDiscoveryFields(t, setupTestBinarySensor, regexpStateTopic))

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		t.Run("OffDelay", testOptional(t, setupTestBinarySensor, func(b *BinarySensor[map[string]any]) {
			b.OffDelay = 5 * time.Minute
		}, `"off_dly"\n300`, `"off_dly"`))
	})
}
