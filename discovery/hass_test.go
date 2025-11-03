package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/hass"
)

func TestHomeAssistantAvailability(t *testing.T) {
	t.Run("Default Prefix", func(t *testing.T) {
		sut := HomeAssistantAvailability(DefaultPrefix)

		require.Equal(t, "homeassistant/status", sut.FullyQualifiedTopic(""))
	})

	t.Run("Custom Prefix", func(t *testing.T) {
		sut := HomeAssistantAvailability("custom")

		require.Equal(t, "custom/status", sut.FullyQualifiedTopic(""))
	})

	t.Run("Unmarshaler", func(t *testing.T) {
		sut := HomeAssistantAvailability(DefaultPrefix)

		_, ok := sut.Get()
		assert.False(t, ok, "should not have a value before first msg")

		sut.ServeMQTT(nil, "homeassistant/status", []byte(hass.Available))
		v, ok := sut.Get()

		assert.True(t, ok, "should have a value after first msg")
		assert.EqualValues(t, hass.Available, v)
	})
}
