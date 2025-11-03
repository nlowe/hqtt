package discovery

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/mqtt"
)

func discardEncoder() *jsontext.Encoder {
	return jsontext.NewEncoder(io.Discard)
}

func capturingEncoder() (*jsontext.Encoder, *bytes.Buffer) {
	b := &bytes.Buffer{}
	return jsontext.NewEncoder(
		b,
		jsontext.AllowDuplicateNames(false),
		jsontext.AllowInvalidUTF8(false),
		jsontext.SpaceAfterComma(false),
		jsontext.SpaceAfterColon(false),
		jsontext.Multiline(false),
	), b
}

func TestDefaultMarshalers(t *testing.T) {
	t.Run("URL as string", func(t *testing.T) {
		e, b := capturingEncoder()

		u, err := url.Parse("http://example.com")
		require.NoError(t, err)

		require.NoError(t, json.MarshalEncode(e, map[string]*url.URL{"sut": u}, json.WithMarshalers(Marshalers)))

		assert.Equal(t, `{"sut":"http://example.com"}`, strings.TrimSpace(b.String()))
	})

	t.Run("Duration as integer seconds", func(t *testing.T) {
		e, b := capturingEncoder()

		d := 5*time.Minute + 42*time.Second + 123*time.Millisecond

		require.NoError(t, json.MarshalEncode(e, map[string]time.Duration{"sut": d}, json.WithMarshalers(Marshalers)))

		assert.Equal(t, `{"sut":342}`, strings.TrimSpace(b.String()))
	})
}

func TestMarshalRequiredTopic(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		require.ErrorIs(
			t,
			MarshalRequiredTopic("sut", discardEncoder(), "", ""),
			ErrTopicRequired,
		)
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MarshalRequiredTopic("", e, "foo", "bar/fizz/buzz"))
		require.EqualValues(t, "\"foo\"\n\"bar/fizz/buzz\"\n", b.String())
	})
}

func TestMarshalRequiredValueTopic(t *testing.T) {
	require.ErrorIs(
		t,
		MarshalRequiredValueTopic[any]("sut", discardEncoder(), "", nil, "bar"),
		ErrTopicRequired,
	)
}

func TestMarshalRequiredRemoteValueTopic(t *testing.T) {
	require.ErrorIs(
		t,
		MarshalRequiredRemoteValueTopic[any]("sut", discardEncoder(), "", nil, "bar"),
		ErrTopicRequired,
	)
}

func TestMaybeMarshalTopic(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalTopic(e, "", ""))
		require.Empty(t, b.Bytes())
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalTopic(e, "foo", "bar/fizz/buzz"))
		require.EqualValues(t, "\"foo\"\n\"bar/fizz/buzz\"\n", b.String())
	})
}

func TestMaybeMarshalValueTopic(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalValueTopic[any](e, "foo", nil, "bar"))
		require.Empty(t, b.Bytes())
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalValueTopic[any](e, "foo", mqtt.NewValue[any]("fizz/buzz", nil), "bar"))
		require.EqualValues(t, "\"foo\"\n\"bar/fizz/buzz\"\n", b.String())
	})
}

func TestMaybeMarshalRemoteValueTopic(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalRemoteValueTopic[any](e, "foo", nil, "bar"))
		require.Empty(t, b.Bytes())
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalRemoteValueTopic[any](e, "foo", mqtt.NewRemoteValue[any]("fizz/buzz", nil), "bar"))
		require.EqualValues(t, "\"foo\"\n\"bar/fizz/buzz\"\n", b.String())
	})
}

func TestMaybeMarshalStateAndCommandTopics(t *testing.T) {
	t.Run("Both Empty", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalStateAndCommandTopics[any]("sut", e, "state", nil, "command", nil, "bar"))
		require.Empty(t, b.Bytes())
	})

	t.Run("One Empty", func(t *testing.T) {
		t.Run("State", func(t *testing.T) {
			e, b := capturingEncoder()

			require.ErrorIs(
				t,
				MaybeMarshalStateAndCommandTopics[any]("sut", e, "state", nil, "command", mqtt.NewRemoteValue[any]("fizz/buzz/command", nil), "bar"),
				ErrMissingStateOrCommandTopic,
			)
			require.Empty(t, b.Bytes())
		})

		t.Run("Command", func(t *testing.T) {
			e, b := capturingEncoder()

			require.ErrorIs(
				t,
				MaybeMarshalStateAndCommandTopics[any]("sut", e, "state", mqtt.NewValue[any]("fizz/buzz/state", nil), "command", nil, "bar"),
				ErrMissingStateOrCommandTopic,
			)
			require.Empty(t, b.Bytes())
		})
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(
			t,
			MaybeMarshalStateAndCommandTopics[any]("sut", e, "state", mqtt.NewValue[any]("fizz/buzz/state", nil), "command", mqtt.NewRemoteValue[any]("fizz/buzz/command", nil), "bar"),
		)

		require.EqualValues(t, `"state"
"bar/fizz/buzz/state"
"command"
"bar/fizz/buzz/command"
`, b.String())
	})
}

func TestMarshalStd(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		e, b := capturingEncoder()

		require.ErrorIs(
			t,
			MarshalStd[int]("sut", e, "foo", nil),
			ErrValueRequired,
		)
		require.Empty(t, b.Bytes())
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		v := 123
		require.NoError(t, MarshalStd[int]("sut", e, "foo", &v))
		require.EqualValues(t, `"foo"
123
`, b.String())
	})
}

func TestMaybeMarshalStd(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalStd[int](e, "foo", nil))
		require.Empty(t, b.Bytes())
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		v := 123
		require.NoError(t, MaybeMarshalStd[int](e, "foo", &v))
		require.EqualValues(t, `"foo"
123
`, b.String())
	})
}

func TestMaybeMarshalStdSlice(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		t.Run("no elements", func(t *testing.T) {
			e, b := capturingEncoder()

			require.NoError(t, MaybeMarshalStdSlice[int](e, "foo", []int{}))
			require.Empty(t, b.Bytes())
		})

		t.Run("nil", func(t *testing.T) {
			e, b := capturingEncoder()

			require.NoError(t, MaybeMarshalStdSlice[int](e, "foo", nil))
			require.Empty(t, b.Bytes())
		})
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalStdSlice[int](e, "foo", []int{123}))
		require.EqualValues(t, `"foo"
[123]
`, b.String())
	})
}

func TestMarshalStdComparable(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		e, b := capturingEncoder()

		var v int

		require.ErrorIs(
			t,
			MarshalStdComparable("sut", e, "foo", v),
			ErrValueRequired,
		)
		require.Empty(t, b.Bytes())
	})

	t.Run("Not Default", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MarshalStdComparable("sut", e, "foo", 123))
		require.EqualValues(t, `"foo"
123
`, b.String())
	})
}

func TestMaybeMarshalStdComparable(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		e, b := capturingEncoder()

		var v int

		require.NoError(t, MaybeMarshalStdComparable(e, "foo", v))
		require.Empty(t, b.Bytes())
	})

	t.Run("Not Default", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeMarshalStdComparable(e, "foo", 123))
		require.EqualValues(t, `"foo"
123
`, b.String())
	})
}

func TestMarshalStdIfNot(t *testing.T) {
	t.Run("Equal", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MarshalStdIfNot(123, e, "foo", 123))
		require.Empty(t, b.Bytes())
	})

	t.Run("Not Equal", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MarshalStdIfNot(123, e, "foo", 456))
		require.EqualValues(t, `"foo"
456
`, b.String())
	})
}

func TestMaybeInlineMarshalStd(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeInlineMarshalStd(e, map[string]string{}))
		require.Empty(t, b.Bytes())
	})

	t.Run("OK", func(t *testing.T) {
		e, b := capturingEncoder()

		require.NoError(t, MaybeInlineMarshalStd(e, map[string]string{"foo": "bar", "fizz": "buzz"}))

		result := b.String()

		assert.Contains(t, result, `"foo"
"bar"
`)
		assert.Contains(t, result, `"fizz"
"buzz"
`)
	})
}
