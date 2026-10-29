package platform

import (
	"encoding/json/jsontext"
	"errors"
	"net/url"

	"github.com/nlowe/hqtt"
	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
)

// UpdateState is the message that is published to an Update's state topic. It holds information about the installed
// version, the latest version, and progress on a running update.
type UpdateState struct {
	// The software or firmware version installed.
	InstalledVersion string `json:"installed_version" hqtt:"required"`
	// The latest software or firmware version available.
	LatestVersion string `json:"latest_version,omitempty"`

	// Title of the software or firmware update available.
	Title string `json:"title,omitempty"`

	// Summary of the software or firmware update available.
	ReleaseSummary string `json:"release_summary,omitempty"`

	// URL pointing to the software release notes.
	ReleaseURL *url.URL `json:"release_url,omitempty"`

	// URL pointing to an image of the update to be applied as entity picture.
	EntityPicture *url.URL `json:"entity_picture,omitempty"`

	// Boolean to report an update is in progress or not.
	InProgress bool `json:"in_progress"`

	// Number between 0 and 100 to report the update process. A nil value resets the in-progress state.
	UpdatePercentage *float64 `json:"update_percentage"`
}

// Update is a hqtt.Platform that implements the update.mqtt integration for Home Assistant.
//
// See https://www.home-assistant.io/integrations/update.mqtt/
type Update struct {
	// The value to publish to the Command topic to trigger an update from home assistant
	InstallPayload string

	// Provides UpdateState to Home Assistant. While not technically required by Home Assistant, hqtt requires this be
	// configured to provide dynamic updates without needing to reconfigure device discovery.
	State *mqtt.Value[UpdateState] `hqtt:"required"`
	// Home Assistant will write InstallPayload to this value when an update is triggered. If not specified, Home
	// Assistant will only show the current and latest version, with no option to install updates.
	Command *mqtt.RemoteValue[string]

	// Information about the device this Update is a part of to tie it into the device registry. Only works when
	// (hqtt.Component).UniqueID is set. At least one of identifiers or connections must be present to identify the
	// device.
	Device *hqtt.Device
	// The kind of software update this entity represents
	DeviceClass hass.UpdateDeviceClass

	// Number of decimal digits for display of update progress.
	DisplayPrecision uint

	// TODO: Encoding? Is this a common option that should be set in the component instead?
}

func (u *Update) PlatformName() string {
	return "update"
}

func (u *Update) Subscriptions(prefix string) []mqtt.Subscription {
	var result []mqtt.Subscription

	result = u.Command.AppendSubscribeOptions(result, prefix)

	return result
}

func (u *Update) ServeMQTT(w mqtt.Writer, topic string, payload []byte) {
	mqtt.Dispatch(
		"",
		u.Command,
	).ServeMQTT(w, topic, payload)
}

func (u *Update) MarshalPlatformDiscoveryTo(e *jsontext.Encoder, prefix string) error {
	if u.Device != nil {
		// When specifying a device link, Home Assistant requires the component's UniqueID is specified. This is
		// required for device-based discovery anyway, so we don't have to check it here. We just need to check that the
		// linked device is valid on the off chance someone specified the device as a struct literal instead of using
		// a pointer to the device that holds the component that holds this platform.

		if err := u.Device.Valid(); err != nil {
			return err
		}
	}

	return errors.Join(
		discovery.MaybeMarshalStdComparable(e, discovery.FieldPayloadInstall, u.InstallPayload),

		discovery.MarshalRequiredValueTopic("state", e, discovery.FieldStateTopic, u.State, prefix),
		discovery.MaybeMarshalRemoteValueTopic(e, discovery.FieldCommandTopic, u.Command, prefix),

		discovery.MaybeMarshalStd(e, discovery.FieldDevice, u.Device),
		discovery.MaybeMarshalStdComparable(e, discovery.FieldDeviceClass, u.DeviceClass),

		discovery.MaybeMarshalStdComparable(e, discovery.FieldDisplayPrecision, u.DisplayPrecision),
	)
}
