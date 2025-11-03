package hqtt

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/mqtt"
)

// ErrInvalidDevice is the error returned by Device.Configure and Device.Valid if it is not properly configured.
var ErrInvalidDevice = errors.New("device must have at least one identifying value in 'identifiers' and/or 'connections'")

// DeviceConnection maps this Device to the outside world. For example:
//
//	DeviceConnection{
//	    Kind: "mac",
//	    Value: "02:5b:26:a8:dc:12",
//	}
//
// It implements fmt.Stringer and slog.LogValuer
type DeviceConnection struct {
	Kind  string
	Value string
}

func (d DeviceConnection) String() string {
	return fmt.Sprintf("[%q,%q]", d.Kind, d.Value)
}

func (d DeviceConnection) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("kind", d.Kind),
		slog.String("value", d.Value),
	)
}

func (d DeviceConnection) MarshalJSONTo(e *jsontext.Encoder) error {
	return errors.Join(
		e.WriteToken(jsontext.BeginArray),
		e.WriteToken(jsontext.String(d.Kind)),
		e.WriteToken(jsontext.String(d.Value)),
		e.WriteToken(jsontext.EndArray),
	)
}

// Device represents an MQTT-based HomeAssistant device. In the Home Assistant MQTT Integration, a Device is a
// collection of "Components" (entities). This relationship is only constructed when marshaling the discovery payload to
// the MQTT Broker.
//
// See https://www.home-assistant.io/integrations/mqtt/#device-discovery-payload
type Device struct {
	// The ID to use for discovery. If empty, an ID is calculated from other fields.
	DiscoveryID string `json:"-"`

	// The name of the device.
	Name string `json:"name,omitempty"`

	// The serial number of the device
	Serial string `json:"sn,omitempty"`

	// The manufacturer of the device.
	Manufacturer string `json:"mf,omitempty"`

	// The model of the device.
	Model string `json:"mdl,omitempty"`

	// The model identifier of the device.
	ModelID string `json:"mdl_id,omitempty"`

	// A link to the webpage that can manage the configuration of this device. Can be either a http://, https:// or an
	// internal homeassistant:// URL.
	ConfigurationURL *url.URL `json:"cu,omitempty"`

	// A list of connections of the device to the outside world. For example, `[]DeviceConnection{{Kind: "mac", Value: "02:5b:26:a8:dc:12"]}}`
	Connections []DeviceConnection `json:"cns,omitempty"`

	// The hardware version of the device.
	HardwareVersion string `json:"hw,omitempty"`

	// The firmware version of the device
	FirmwareVersion string `json:"sw,omitempty"`

	// A list of IDs that uniquely identify the device. For example a serial number.
	Identifiers []string `json:"ids,omitempty"`

	// Suggest an area if the devic e isn't in one yet
	SuggestedArea string `json:"sa,omitempty"`

	// It is recommended to add information about the origin of MQTT entities. The origin details will be logged in the
	// core event log when an item is discovered or updated. Adding origin information helps with troubleshooting and
	// provides valuable context about the source of MQTT messages in your Home Assistant setup. Home Assistant requires
	// origin information be specified when using Device-based Discovery. If omitted, DefaultOrigin will be used when
	// serializing the discovery payload instead.
	Origin *Origin `json:"-"`

	// Identifier of a device that routes messages between this device and Home Assistant. Examples of such devices are
	// hubs, or parent devices of a sub-device. This is used to show device topology in Home Assistant.
	ViaDevice string `json:"via_device,omitempty"`
}

// ID calculates an identifier for this device. If the Device.DiscoveryID is specified, that value will be used.
// Otherwise, if any of the following fields are set, they are used (separated by discovery.IDSep): All
// Device.Identifiers, Device.Name, Device.Serial, Device.Manufacturer, Device.Model, and Device.ModelID.
func (d *Device) ID() string {
	if d.DiscoveryID != "" {
		return d.DiscoveryID
	}

	var result strings.Builder

	writeSep := func() {
		if result.Len() > 0 {
			result.WriteString(discovery.IDSep)
		}
	}

	if len(d.Identifiers) > 0 {
		for i, ident := range d.Identifiers {
			result.WriteString(discovery.IDSanitizer.Replace(ident))
			if i < len(d.Identifiers)-1 {
				writeSep()
			}
		}
	}

	if d.Name != "" {
		writeSep()
		result.WriteString(discovery.IDSanitizer.Replace(d.Name))
	}

	if d.Serial != "" {
		writeSep()
		result.WriteString(discovery.IDSanitizer.Replace(d.Serial))
	}

	if d.Manufacturer != "" {
		writeSep()
		result.WriteString(discovery.IDSanitizer.Replace(d.Manufacturer))
	}

	if d.Model != "" {
		writeSep()
		result.WriteString(discovery.IDSanitizer.Replace(d.Model))
	}

	if d.ModelID != "" {
		writeSep()
		result.WriteString(discovery.IDSanitizer.Replace(d.ModelID))
	}

	return result.String()
}

// Valid checks if this Device is configured appropriately. Home Assistant requires at least one value be configured for
// Device.Identifiers, or at least one value be configured for Device.Connections.
func (d *Device) Valid() error {
	if len(d.Identifiers) == 0 && len(d.Connections) == 0 {
		return ErrInvalidDevice
	}

	return nil
}

// Configure updates the device discovery payload for this device and the provided components, which are associated with
// this Device. To remove components from the device, replace the component in the map with a RemoveComponent when
// calling Configure.
//
// The device must pass validation performed by Device.Valid.
func (d *Device) Configure(ctx context.Context, w mqtt.Writer, discoveryPrefix string, components map[string]json.MarshalerTo) error {
	// Validation
	if err := d.Valid(); err != nil {
		return err
	}

	// Write Device
	var buf bytes.Buffer
	e := jsontext.NewEncoder(
		&buf,
		jsontext.CanonicalizeRawInts(true),
		jsontext.CanonicalizeRawFloats(true),
	)

	err := errors.Join(
		e.WriteToken(jsontext.BeginObject),

		discovery.MarshalStd("device", e, discovery.FieldDevice, d),
		discovery.MarshalStd("origin", e, discovery.FieldOrigin, cmp.Or(d.Origin, &DefaultOrigin)),

		e.WriteToken(jsontext.String(discovery.FieldComponents)),
		e.WriteToken(jsontext.BeginObject),

		discovery.MaybeInlineMarshalStd(e, components),

		e.WriteToken(jsontext.EndObject),
		// TODO: Shared QoS?
		e.WriteToken(jsontext.EndObject),
	)

	if err != nil {
		return fmt.Errorf("configure: marshal discovery config: %w", err)
	}

	topic := fmt.Sprintf(`%s/device/%s/config`, discoveryPrefix, d.ID())
	return w.WriteTopic(ctx, topic, mqtt.WriteOptions{Retain: true}, buf.Bytes())
}
