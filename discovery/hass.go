package discovery

import (
	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

const (
	// DefaultPrefix is the MQTT Topic Prefix that Home Assistant looks for discovery payloads under
	DefaultPrefix = "homeassistant"
	// StatusTopic is the MQTT Topic that Home Assistant publishes hass.Availability state to for itself.
	StatusTopic = "status"
)

// HomeAssistantAvailability constructs a mqtt.RemoteValue that monitor's Home Assistant's availability topic. Subscribe
// to changes to this value to be notified when Home Assistant restarts.
//
// See https://www.home-assistant.io/integrations/mqtt/#birth-and-last-will-messages.
func HomeAssistantAvailability(discoveryPrefix string) *mqtt.RemoteValue[hass.Availability] {
	return mqtt.NewRemoteValue(mqtt.JoinTopic(discoveryPrefix, StatusTopic), hass.AvailabilityUnmarshaler)
}
