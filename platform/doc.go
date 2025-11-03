// Package platform contains implementations for various Home Assistant MQTT platforms. See the Home Assistant docs for
// a list of supported platforms: https://www.home-assistant.io/integrations/mqtt. Note: Not all platforms may be
// implemented by hqtt.
//
// Each hqtt platform implementation satisfies the hqtt.Platform interface. The PlatformName method returns the Home
// Assistant platform name (e.g. Light's PlatformName method returns the string "light").
//
// Not all fields for a given platform implementation are required by Home Assistant. Required fields will be tagged
// with `hqtt:"required"`. and checked when marshaling for discovery.
package platform
