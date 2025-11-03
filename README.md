# hqtt - Go SDK for writing Home Assistant MQTT Integrations

[![Go Reference](https://pkg.go.dev/badge/github.com/nlowe/hqtt.svg)](https://pkg.go.dev/github.com/nlowe/hqtt) [![](https://github.com/nlowe/hqtt/workflows/CI/badge.svg)](https://github.com/nlowe/hqtt/actions) [![Coverage Status](https://coveralls.io/repos/github/nlowe/hqtt/badge.svg?branch=master)](https://coveralls.io/github/nlowe/hqtt?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/nlowe/hqtt)](https://goreportcard.com/report/github.com/nlowe/hqtt) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](./LICENSE)

> [!IMPORTANT]
> This SDK Requires Go 1.25+. If using Go 1.25, the jsonv2 GOEXPERIMENT must be enabled.

> [!WARNING]
> This SDK is still under active development. The API probably ***will*** change. Feel free to experiment but expect
> missing features, bugs, and functionality to break as development continues.

The SDK is designed around [Home Assistant MQTT Device Discovery](https://www.home-assistant.io/integrations/mqtt/#device-discovery-payload)
via the [`Device`](https://pkg.go.dev/github.com/nlowe/hqtt#Device) type. Devices contain one or more
[`Component[T Platform]`](https://pkg.go.dev/github.com/nlowe/hqtt#Component) components which map to entities provided
by the device. The [`Platform`](https://pkg.go.dev/github.com/nlowe/hqtt#Platform) interface provides basic platform
information that is required to form the Device Discovery payload:

```go
package hqtt

// Platform is the interface implemented by every MQTT Entity Component type.
type Platform interface {
    mqtt.Handler

    // MarshalDiscoveryTo marshals MQTT Device Discovery information to the specified jsontext.Encoder using the
    // provided prefix for all MQTT Topics.
    MarshalDiscoveryTo(e *jsontext.Encoder, prefix string) error

    // PlatformName returns the value for the `platform` field when configuring a component using this platform for MQTT
    // Device Discovery.
    PlatformName() string

    // Subscriptions returns the set of paho.SubscribeOptions for configured fields of this component. Only fields that
    // are properly configured should be included. Typically, each subscription is individually subscribed to, but other
    // mqtt.Subscriber implementations may choose to group topics with wildcards.
    Subscriptions(prefix string) []mqtt.Subscription
}
```

The following platforms are currently implemented:

* [`binary_sensor`](https://www.home-assistant.io/integrations/binary_sensor.mqtt/): [`platform.BinarySensor[TAttributes any]`](https://pkg.go.dev/github.com/nlowe/hqtt/platform#BinarySensor)
* [`light`](https://www.home-assistant.io/integrations/light.mqtt/): [`platform.Light`](https://pkg.go.dev/github.com/nlowe/hqtt/platform#Light)
* [`sensor`](https://www.home-assistant.io/integrations/sensor.mqtt/): [`platform.Sensor[TValue, TAttributes any]`](https://pkg.go.dev/github.com/nlowe/hqtt/platform#Sensor)

You can create a `Component` for other MQTT Entity types by providing a type that satisfies the `Platform` interface,
but if you find yourself implementing a core MQTT Entity type provided by Home Assistant please send a pull request to
add it to the SDK!

The [`discovery` package](https://pkg.go.dev/github.com/nlowe/hqtt/discovery) provides helpers for constructing minified
Device Discovery payloads (including constants for abbreviated field keys). Unless you are implementing support for a
new platform you will typically not need to import this package.

This package also contains a helper for watching the state of Home Assistant itself (which it publishes to
`homeassistant/status` by default).

A subset of Home Assistant core types (e.g. `Availability`, Power/Switch state, etc.) are provided by the
[`hass` package](https://pkg.go.dev/github.com/nlowe/hqtt/hass).

HQTT utilizes [`log/slog`](https://pkg.go.dev/log/slog) for logging. See the
[`log` package](https://pkg.go.dev/github.com/nlowe/hqtt/log) package for details on configuring logging for the SDK.

## MQTT Client Support

MQTT abstractions are provided by the [`mqtt` package](https://pkg.go.dev/github.com/nlowe/hqtt/mqtt). The two main
types you will interact with are [`Value[T any]`](https://pkg.go.dev/github.com/nlowe/hqtt/mqtt#Value) (for values that
should be written to an MQTT topic), and [`RemoteValute[T any]`](https://pkg.go.dev/github.com/nlowe/hqtt/mqtt#Value)
(for values to be read from MQTT Subscriptions).

Values require a [`ValueMarshaler[T]`](https://pkg.go.dev/github.com/nlowe/hqtt/mqtt#ValueMarshaler) which is used to
encode the value when writing to MQTT. Typically, this is a [`StringMarshaler`](https://pkg.go.dev/github.com/nlowe/hqtt/mqtt#StringMarshaler)
(which encodes the value as a string), or a [`JsonValueMarshaler`](https://pkg.go.dev/github.com/nlowe/hqtt/mqtt#JsonValueMarshaler)
(which encodes the value using its JSON representation).

RemoteValues require a [`ValueUnmarshaler[T]`](https://pkg.go.dev/github.com/nlowe/hqtt/mqtt#ValueUnmarshaler), which
does the same thing but in reverse.

Right now, the only officially supported client is [`github.com/eclipse/paho.golang/autopaho`](https://pkg.go.dev/github.com/eclipse/paho.golang/autopaho),
a v5 MQTT client by Eclipse. You can implement your own adapter for any client by implementing the following interfaces
from the [`mqtt`](https://pkg.go.dev/nlowe/hqtt/mqtt) package:

```go
package mqtt

// Writer is the minimum abstraction around writing values to MQTT.
type Writer interface {
	// WriteTopic writes the provided value to the specified topic with the specified WriteOptions.
	WriteTopic(ctx context.Context, topic string, options WriteOptions, value []byte) error
}

// Subscriber manages MQTT Subscriptions
type Subscriber interface {
    // Subscribe configures the underlying MQTT connection to send the client messages for the provided subscriptions.
    // The provided Handler will be called for all subscribed topics in this call.
    Subscribe(ctx context.Context, handler Handler, subscriptions ...Subscription) error

    // Unsubscribe removes any subscriptions configured for the specified topics.
    Unsubscribe(ctx context.Context, topics ...string) error
}
```

See [`example/fake_light`](./example/fake_light) for a small example.
