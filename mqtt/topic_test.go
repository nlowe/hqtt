package mqtt

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimTopic(t *testing.T) {
	for _, tt := range []struct {
		topic string
		want  string
	}{
		{topic: "", want: ""},
		{topic: "/", want: ""},
		{topic: "/a", want: "a"},
		{topic: "a/", want: "a"},
		{topic: "/a/", want: "a"},
		{topic: "/a/b", want: "a/b"},
		{topic: "a/b/", want: "a/b"},
		{topic: "a/b", want: "a/b"},
		{topic: "/a/b/", want: "a/b"},
	} {
		t.Run(tt.topic, func(t *testing.T) {
			require.Equal(t, tt.want, TrimTopic(tt.topic))
		})
	}
}

func TestJoinTopic(t *testing.T) {
	for i, tt := range []struct {
		parts []string
		want  string
	}{
		// JoinTopic should trim empty parts
		{parts: []string{""}, want: ""},
		{parts: []string{"", ""}, want: ""},
		{parts: []string{"", "a"}, want: "a"},
		{parts: []string{"", "a", "", "b"}, want: "a/b"},

		// JoinTopic should trim each individual part
		{parts: []string{"a", "/", "b"}, want: "a/b"},
		{parts: []string{"/a", "b"}, want: "a/b"},
		{parts: []string{"a/", "b"}, want: "a/b"},
		{parts: []string{"/a/", "b"}, want: "a/b"},
		{parts: []string{"/a/b", "c"}, want: "a/b/c"},
		{parts: []string{"a/b/", "c"}, want: "a/b/c"},
		{parts: []string{"a/b", "c"}, want: "a/b/c"},
		{parts: []string{"/a/b/", "c"}, want: "a/b/c"},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			require.Equal(t, tt.want, JoinTopic(tt.parts...))
		})
	}
}
