package hqtt

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/hass"
	"github.com/nlowe/hqtt/mqtt"
	"github.com/nlowe/hqtt/mqtt/adapter/testqtt"
)

func setupTestComponent(t *testing.T) (*Component[*testPlatform], *testPlatform, *testqtt.Broker) {
	t.Helper()

	p := &testPlatform{}

	return &Component[*testPlatform]{
		Platform:     p,
		TopicPrefix:  "sut",
		Availability: mqtt.NewValue[hass.Availability]("available", hass.AvailabilityMarshaler),
		UniqueID:     "test",
	}, p, &testqtt.Broker{}
}

func testRequired(t *testing.T, omit func(c *Component[*testPlatform]), okRegexp, absentErr string) func(*testing.T) {
	t.Helper()

	return func(t *testing.T) {
		t.Helper()
		t.Parallel()

		t.Run("Ok", func(t *testing.T) {
			t.Helper()
			t.Parallel()

			sut, _, _ := setupTestComponent(t)

			var b bytes.Buffer
			e := jsontext.NewEncoder(&b)

			require.NoError(t, sut.MarshalComponentDiscoveryTo(e))

			require.Regexp(t, okRegexp, b.String())
		})

		t.Run("Missing", func(t *testing.T) {
			t.Helper()
			t.Parallel()

			sut, _, _ := setupTestComponent(t)
			omit(sut)

			var b bytes.Buffer
			e := jsontext.NewEncoder(&b)

			require.EqualError(t, sut.MarshalComponentDiscoveryTo(e), absentErr)
		})
	}
}

func testOptional(t *testing.T, modify func(c *Component[*testPlatform]), presentRegexp, absentRegexp string) func(*testing.T) {
	t.Helper()

	return func(t *testing.T) {
		t.Helper()
		t.Parallel()

		t.Run("Present", func(t *testing.T) {
			t.Helper()
			t.Parallel()

			sut, _, _ := setupTestComponent(t)
			modify(sut)

			var b bytes.Buffer
			e := jsontext.NewEncoder(&b)

			require.NoError(t, sut.MarshalComponentDiscoveryTo(e))

			require.Regexp(t, presentRegexp, b.String())
		})

		t.Run("Absent", func(t *testing.T) {
			t.Helper()
			t.Parallel()

			sut, _, _ := setupTestComponent(t)

			var b bytes.Buffer
			e := jsontext.NewEncoder(&b)

			require.NoError(t, sut.MarshalComponentDiscoveryTo(e))

			require.NotRegexp(t, absentRegexp, b.String())
		})
	}
}

func TestComponent_ForRemoval(t *testing.T) {
	sut, _, _ := setupTestComponent(t)

	remove := sut.ForRemoval()

	require.Equal(t, sut.Platform.PlatformName(), remove.Platform)

	var b bytes.Buffer
	e := jsontext.NewEncoder(&b)

	require.NoError(t, remove.MarshalComponentDiscoveryTo(e))
	require.Equal(t, "{\"platform\":\"test\"}\n", b.String())
}

func TestComponent_Subscribe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx := t.Context()

		sut, p, b := setupTestComponent(t)
		c := b.Connect()

		p.SubscriptionConfig = []mqtt.Subscription{
			{Topic: "a"},
			{Topic: "b"},
			{Topic: "c"},
		}

		require.NoError(t, sut.Subscribe(ctx, c))

		require.NoError(t, b.WriteTopic(ctx, "sut/a", mqtt.WriteOptions{}, []byte{'a'}))
		require.NoError(t, b.WriteTopic(ctx, "sut/b", mqtt.WriteOptions{}, []byte{'b'}))
		require.NoError(t, b.WriteTopic(ctx, "sut/c", mqtt.WriteOptions{}, []byte{'c'}))

		p.AssertMessage(t, "a", []byte{'a'})
		p.AssertMessage(t, "b", []byte{'b'})
		p.AssertMessage(t, "c", []byte{'c'})
	})

	t.Run("error", func(t *testing.T) {
		t.Run("ErrComponentAlreadySubscribed", func(t *testing.T) {
			sut, p, b := setupTestComponent(t)
			c := b.Connect()

			p.SubscriptionConfig = []mqtt.Subscription{
				{Topic: "a"},
				{Topic: "b"},
				{Topic: "c"},
			}

			require.NoError(t, sut.Subscribe(t.Context(), c))
			require.ErrorIs(t, sut.Subscribe(t.Context(), c), ErrComponentAlreadySubscribed)
		})

		t.Run("Subscribe", func(t *testing.T) {
			sut, p, b := setupTestComponent(t)
			c := b.Connect()

			p.SubscriptionConfig = []mqtt.Subscription{
				{Topic: "a"},
				{Topic: "b"},
				{Topic: "c"},
			}

			c.RejectSubscribeWith(fmt.Errorf("dummy"))
			require.EqualError(t, sut.Subscribe(t.Context(), c), "dummy")
		})
	})
}

func TestComponent_Unsubscribe(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx := t.Context()

		sut, p, b := setupTestComponent(t)
		c := b.Connect()

		p.SubscriptionConfig = []mqtt.Subscription{
			{Topic: "a"},
			{Topic: "b"},
			{Topic: "c"},
		}

		require.NoError(t, sut.Subscribe(ctx, c))
		require.NoError(t, sut.Unsubscribe(ctx, c))

		require.NoError(t, b.WriteTopic(ctx, "sut/a", mqtt.WriteOptions{}, []byte{'a'}))
		require.NoError(t, b.WriteTopic(ctx, "sut/b", mqtt.WriteOptions{}, []byte{'b'}))
		require.NoError(t, b.WriteTopic(ctx, "sut/c", mqtt.WriteOptions{}, []byte{'c'}))

		require.Empty(t, p.All())
	})

	t.Run("NotSubscribed", func(t *testing.T) {
		sut, _, b := setupTestComponent(t)
		c := b.Connect()

		c.RejectUnsubscribeWith(fmt.Errorf("this should not be called"))

		require.NoError(t, sut.Unsubscribe(t.Context(), c))
	})

	t.Run("error", func(t *testing.T) {
		ctx := t.Context()

		sut, p, b := setupTestComponent(t)
		c := b.Connect()

		p.SubscriptionConfig = []mqtt.Subscription{
			{Topic: "a"},
			{Topic: "b"},
			{Topic: "c"},
		}

		require.NoError(t, sut.Subscribe(ctx, c))

		c.RejectUnsubscribeWith(fmt.Errorf("dummy"))
		require.EqualError(t, sut.Unsubscribe(ctx, c), "dummy")
	})
}

//nolint:goconst
func TestComponent_MarshalJSONTo(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		t.Parallel()

		t.Run("Availability", testRequired(t, func(c *Component[*testPlatform]) {
			c.Availability = nil
		}, `"avty_t":"sut/available"`, "availability: topic is required"))

		t.Run("UniqueID", testRequired(t, func(c *Component[*testPlatform]) {
			c.UniqueID = ""
		}, `"uniq_id":"[\w-]+"`, "UniqueID: value is required"))
	})

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		t.Run("EntityCategory", testOptional(t, func(c *Component[*testPlatform]) {
			c.EntityCategory = "foo"
		}, `"ent_cat":"foo"`, `"ent_cat"`))

		t.Run("Icon", testOptional(t, func(c *Component[*testPlatform]) {
			c.Icon = "mdi:plug"
		}, `"ic":"mdi:plug"`, `"ic"`))

		t.Run("Picture", testOptional(t, func(c *Component[*testPlatform]) {
			var err error
			c.Picture, err = url.Parse("https://example.com/foo.png")
			require.NoError(t, err)
		}, `"picture":"https://example.com/foo.png"`, `"picture"`))

		t.Run("CustomAvailability", func(t *testing.T) {
			t.Parallel()

			t.Run("PayloadAvailable", testOptional(t, func(c *Component[*testPlatform]) {
				c.CustomAvailabilityValues.Available = "foo"
			}, `"pl_avail":"foo"`, `"pl_avail"`))
			t.Run("PayloadUnavailable", testOptional(t, func(c *Component[*testPlatform]) {
				c.CustomAvailabilityValues.Unavailable = "bar"
			}, `"pl_not_avail":"bar"`, `"pl_not_avail"`))
		})

		t.Run("DefaultEntityID", testOptional(t, func(c *Component[*testPlatform]) {
			c.DefaultEntityID = "example"
		}, `"def_ent_id":"example"`, `"def_ent_id"`))

		t.Run("WriteOptions", func(t *testing.T) {
			t.Parallel()

			t.Run("QoS", testOptional(t, func(c *Component[*testPlatform]) {
				c.WriteOptions.QoS = mqtt.QOSExactlyOnce
			}, `"qos":2`, `"qos"`))

			t.Run("Retain", testOptional(t, func(c *Component[*testPlatform]) {
				c.WriteOptions.Retain = true
			}, `"ret":true`, `"retain"`))
		})
	})

	t.Run("Name", func(t *testing.T) {
		t.Run("Empty", func(t *testing.T) {
			sut, _, _ := setupTestComponent(t)

			var b bytes.Buffer
			e := jsontext.NewEncoder(&b)

			require.NoError(t, sut.MarshalComponentDiscoveryTo(e))

			result := b.String()
			assert.Regexp(t, `"name":null`, result)
		})

		t.Run("Ok", func(t *testing.T) {
			sut, _, _ := setupTestComponent(t)
			sut.Name = "testComponent"

			var b bytes.Buffer
			e := jsontext.NewEncoder(&b)

			require.NoError(t, sut.MarshalComponentDiscoveryTo(e))

			result := b.String()
			assert.Regexp(t, `"name":"testComponent"`, result)
		})
	})

	t.Run("Platform Error", func(t *testing.T) {
		sut, p, _ := setupTestComponent(t)
		var b bytes.Buffer
		e := jsontext.NewEncoder(&b)

		p.MarshalError = fmt.Errorf("dummy")

		require.EqualError(t, sut.MarshalComponentDiscoveryTo(e), "dummy")
	})
}
