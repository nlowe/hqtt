package platform

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/discovery"

	"github.com/nlowe/hqtt/mqtt"
)

const regexpStateTopic = `"stat_t"\n"sut/state"`

func testRequiredDiscoveryFields[T discovery.PlatformMarshaler](t *testing.T, factory func(*testing.T) (T, *jsontext.Encoder, *bytes.Buffer), requiredRegexp ...string) func(*testing.T) {
	t.Helper()

	return func(t *testing.T) {
		sut, w, b := factory(t)

		require.NoError(t, sut.MarshalPlatformDiscoveryTo(w, "sut"))

		for _, pattern := range requiredRegexp {
			assert.Regexp(t, pattern, b.String())
		}
	}
}

func testOptional[T discovery.PlatformMarshaler](t *testing.T, factory func(*testing.T) (T, *jsontext.Encoder, *bytes.Buffer), modify func(T), presentRegexp, absentRegexp string) func(*testing.T) {
	t.Helper()

	return func(t *testing.T) {
		t.Parallel()

		t.Run("Present", func(t *testing.T) {
			t.Parallel()

			sut, w, b := factory(t)
			modify(sut)

			require.NoError(t, sut.MarshalPlatformDiscoveryTo(w, "sut"))
			assert.Regexp(t, presentRegexp, b.String())
		})

		t.Run("Absent", func(t *testing.T) {
			t.Parallel()

			sut, w, b := factory(t)

			require.NoError(t, sut.MarshalPlatformDiscoveryTo(w, "sut"))
			assert.NotRegexp(t, absentRegexp, b.String())
		})
	}
}

func assertHasTopicSubscription(t *testing.T, subscriptions []mqtt.Subscription, topic string) {
	t.Helper()

	for _, sub := range subscriptions {
		if sub.Topic == topic {
			return
		}
	}

	t.Errorf("topic %q not found in subscriptions: %v", topic, subscriptions)
}

func assertHasRemoteValue[T any](t *testing.T, v *mqtt.RemoteValue[T], expected T) {
	t.Helper()

	vv, ok := v.Get()

	if assert.True(t, ok, "expected a value to be written") {
		assert.EqualValues(t, expected, vv)
	}
}
