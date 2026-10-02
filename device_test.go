package hqtt

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/mqtt/adapter/testqtt"

	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/mqtt"
)

//nolint:goconst
func TestDevice_ID(t *testing.T) {
	t.Run("Explicit", func(t *testing.T) {
		sut := &Device{DiscoveryID: "foo"}

		require.Equal(t, "foo", sut.ID())
	})

	t.Run("Identifiers", func(t *testing.T) {
		sut := &Device{Identifiers: []string{"A", "B", "C"}}

		require.Equal(t, "A__B__C", sut.ID())
	})

	t.Run("Name", func(t *testing.T) {
		sut := &Device{Name: "foo"}

		require.Equal(t, "foo", sut.ID())
	})

	t.Run("Serial", func(t *testing.T) {
		sut := &Device{Serial: "foo"}

		require.Equal(t, "foo", sut.ID())
	})

	t.Run("Manufacturer", func(t *testing.T) {
		sut := &Device{Manufacturer: "foo"}

		require.Equal(t, "foo", sut.ID())
	})

	t.Run("Model", func(t *testing.T) {
		sut := &Device{Model: "foo"}

		require.Equal(t, "foo", sut.ID())
	})

	t.Run("ModelID", func(t *testing.T) {
		sut := &Device{ModelID: "foo"}

		require.Equal(t, "foo", sut.ID())
	})

	t.Run("Combined", func(t *testing.T) {
		sut := &Device{
			Name:         "name",
			Serial:       "serial",
			Manufacturer: "manufacturer",
			Model:        "model",
			ModelID:      "model_id",
			Identifiers:  []string{"id_1", "id:2", "id=3"},
		}

		require.Equal(t, "id_1__id_2__id_3__name__serial__manufacturer__model__model_id", sut.ID())
	})
}

func TestDevice_Valid(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		t.Run("Identifiers", func(t *testing.T) {
			sut := &Device{Identifiers: []string{"A", "B", "C"}}

			require.NoError(t, sut.Valid())
		})

		t.Run("Connections", func(t *testing.T) {
			sut := &Device{Connections: []DeviceConnection{{
				Kind:  "mac",
				Value: "aa:bb:cc:dd:11:22:33:44",
			}}}

			require.NoError(t, sut.Valid())
		})
	})

	t.Run("Error", func(t *testing.T) {
		configurationURL, err := url.Parse("http://foo/confiig")
		require.NoError(t, err)

		supportURL, err := url.Parse("http://foo/support")
		require.NoError(t, err)

		sut := &Device{
			Connections: nil,
			Identifiers: nil,

			DiscoveryID:      "DiscoveryID",
			Name:             "Name",
			Serial:           "Serial",
			Manufacturer:     "Manufacturer",
			Model:            "Model",
			ModelID:          "ModelID",
			ConfigurationURL: configurationURL,
			HardwareVersion:  "HardwareVersion",
			FirmwareVersion:  "FirmwareVersion",
			SuggestedArea:    "SuggestedArea",
			Origin: &Origin{
				Name:            "OriginName",
				SoftwareVersion: "SoftwareVersion",
				SupportURL:      supportURL,
			},
			ViaDevice: "ViaDevice",
		}

		require.ErrorIs(t, sut.Valid(), ErrInvalidDevice)
	})
}

//nolint:goconst
func TestDevice_Configure(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		t.Run("With Origin", func(t *testing.T) {
			configurationURL, err := url.Parse("http://foo/confiig")
			require.NoError(t, err)

			supportURL, err := url.Parse("http://foo/support")
			require.NoError(t, err)

			sut := &Device{
				DiscoveryID:      "example",
				Name:             "Name",
				Serial:           "Serial",
				Manufacturer:     "Manufacturer",
				Model:            "Model",
				ModelID:          "ModelID",
				ConfigurationURL: configurationURL,
				HardwareVersion:  "HardwareVersion",
				FirmwareVersion:  "FirmwareVersion",
				SuggestedArea:    "SuggestedArea",
				Origin: &Origin{
					Name:            "OriginName",
					SoftwareVersion: "SoftwareVersion",
					SupportURL:      supportURL,
				},
				ViaDevice:   "ViaDevice",
				Identifiers: []string{"A", "B", "C"},
			}

			c, _, b := setupTestComponent(t)

			conn := b.Connect()

			require.NoError(
				t,
				sut.Configure(
					t.Context(),
					conn,
					mqtt.WriteOptions{},
					discovery.DefaultPrefix,
					map[string]discovery.ComponentMarshaler{
						"example": c,
					},
				),
			)

			conn.Sent.AssertMessage(
				t,
				`homeassistant/device/example/config`,
				[]byte(`{"dev":{"name":"Name","sn":"Serial","mf":"Manufacturer","mdl":"Model","mdl_id":"ModelID","cu":"http://foo/confiig","hw":"HardwareVersion","sw":"FirmwareVersion","ids":["A","B","C"],"sa":"SuggestedArea","via_device":"ViaDevice"},"o":{"name":"OriginName","sw":"SoftwareVersion","url":"http://foo/support"},"cmps":{"example":{"p":"test","name":null,"avty_t":"sut/available","uniq_id":"test","platform_topic_prefix":"sut"}}}`+"\n"),
			)
		})

		t.Run("Default Origin", func(t *testing.T) {
			sut := &Device{
				DiscoveryID: "example",
				Identifiers: []string{"A", "B", "C"},
			}

			c, _, b := setupTestComponent(t)

			conn := b.Connect()

			require.NoError(
				t,
				sut.Configure(
					t.Context(),
					conn,
					mqtt.WriteOptions{},
					discovery.DefaultPrefix,
					map[string]discovery.ComponentMarshaler{
						"example": c,
					},
				),
			)

			conn.Sent.AssertMessage(
				t,
				`homeassistant/device/example/config`,
				[]byte(`{"dev":{"ids":["A","B","C"]},"o":{"name":"hqtt","sw":"master","url":"https://github.com/nlowe/hqtt"},"cmps":{"example":{"p":"test","name":null,"avty_t":"sut/available","uniq_id":"test","platform_topic_prefix":"sut"}}}`+"\n"),
			)
		})
	})

	t.Run("Error", func(t *testing.T) {
		t.Run("Invalid", func(t *testing.T) {
			sut := &Device{}

			require.ErrorIs(
				t,
				sut.Configure(t.Context(), nil, mqtt.WriteOptions{}, discovery.DefaultPrefix, nil),
				ErrInvalidDevice,
			)
		})

		t.Run("Marshal", func(t *testing.T) {
			sut := &Device{Identifiers: []string{"A", "B", "C"}}
			c, p, b := setupTestComponent(t)
			p.MarshalError = fmt.Errorf("dummy")

			conn := b.Connect()

			require.EqualError(
				t,
				sut.Configure(
					t.Context(),
					conn,
					mqtt.WriteOptions{},
					discovery.DefaultPrefix,
					map[string]discovery.ComponentMarshaler{
						"example": c,
					},
				),
				"configure: marshal discovery config: dummy",
			)
		})

		t.Run("Write", func(t *testing.T) {
			b := &testqtt.Broker{}
			conn := b.Connect()

			conn.RejectWriteWith(fmt.Errorf("dummy"))

			sut := &Device{Identifiers: []string{"A", "B", "C"}}

			require.EqualError(
				t,
				sut.Configure(t.Context(), conn, mqtt.WriteOptions{}, discovery.DefaultPrefix, nil),
				"dummy",
			)
		})
	})
}
