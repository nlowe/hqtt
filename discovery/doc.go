// Package discovery contains constants and utilities for constructing Home Assistant Device Discovery MQTT Payloads. To
// minimize throughput to MQTT (and to minimize storage used for retained messages), the constants in this package map
// to the abbreviated forms. See the Home Assistant documentation for a full list of abbreviations.
//
// See https://www.home-assistant.io/integrations/mqtt/#supported-abbreviations-in-mqtt-discovery-messages for a full
// list of abbreviations. Not all abbreviations are provided as constants by this package.
package discovery
