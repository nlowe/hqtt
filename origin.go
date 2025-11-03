package hqtt

import "net/url"

// Origin provides information about the software providing devices over MQTT to Home Assistant. See the documentation
// for Device.Origin for details.
type Origin struct {
	// The name of the application that is the origin of the discovered MQTT item.
	Name string `json:"name"`
	// Software version of the application that supplies the discovered MQTT item.
	SoftwareVersion string `json:"sw,omitempty"`
	// Support URL of the application that supplies the discovered MQTT item.
	SupportURL *url.URL `json:"url,omitempty"`
}

var (
	hqttSupportUrl, _ = url.Parse("https://github.com/nlowe/hqtt")

	// DefaultOrigin provides origin information to Home Assistant for applications that do not otherwise specify one.
	DefaultOrigin = Origin{
		Name:            "hqtt",
		SoftwareVersion: "master",
		SupportURL:      hqttSupportUrl,
	}
)
