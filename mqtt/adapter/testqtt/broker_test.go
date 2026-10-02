package testqtt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nlowe/hqtt/mqtt"
)

func TestBroker(t *testing.T) {
	sut := &Broker{}

	a := sut.Connect()
	b := sut.Connect()
	c := sut.Connect()

	ctx := t.Context()

	var aHandler, bHandler, cHandler int

	require.NoError(t, sut.WriteTopic(ctx, "for/a/foo", mqtt.WriteOptions{}, []byte{0}))
	require.NoError(t, sut.WriteTopic(ctx, "for/b/foo", mqtt.WriteOptions{}, []byte{0}))
	require.NoError(t, sut.WriteTopic(ctx, "for/c/foo", mqtt.WriteOptions{}, []byte{0}))

	assert.Empty(t, a.Seen.All())
	assert.Empty(t, b.Seen.All())
	assert.Empty(t, c.Seen.All())

	require.NoError(t, a.Subscribe(ctx, mqtt.HandlerFunc(func(w mqtt.Writer, s string, v []byte) {
		aHandler++
	}), mqtt.Subscription{Topic: "for/a/#"}, mqtt.Subscription{Topic: "homeassistant/status"}))

	require.NoError(t, b.Subscribe(ctx, mqtt.HandlerFunc(func(w mqtt.Writer, s string, v []byte) {
		bHandler++
	}), mqtt.Subscription{Topic: "for/b/#"}))

	require.NoError(t, c.Subscribe(ctx, mqtt.HandlerFunc(func(w mqtt.Writer, s string, v []byte) {
		cHandler++
	}), mqtt.Subscription{Topic: "for/c/#"}))

	require.NoError(t, sut.WriteTopic(ctx, "for/a/foo", mqtt.WriteOptions{}, []byte{1}))
	require.NoError(t, sut.WriteTopic(ctx, "for/b/foo", mqtt.WriteOptions{}, []byte{1}))
	require.NoError(t, sut.WriteTopic(ctx, "for/c/foo", mqtt.WriteOptions{}, []byte{1}))

	a.Seen.AssertMessage(t, "for/a/foo", []byte{1})
	b.Seen.AssertMessage(t, "for/b/foo", []byte{1})
	c.Seen.AssertMessage(t, "for/c/foo", []byte{1})

	require.NoError(t, sut.WriteTopic(ctx, "homeassistant/status", mqtt.WriteOptions{}, []byte("available")))

	a.Seen.AssertAnyMessage(t, "homeassistant/status")
	b.Seen.AssertNoMessage(t, "homeassistant/status")
	c.Seen.AssertNoMessage(t, "homeassistant/status")

	sut.Disconnect(b)

	require.NoError(t, sut.WriteTopic(ctx, "for/a/foo", mqtt.WriteOptions{}, []byte{2}))
	require.NoError(t, sut.WriteTopic(ctx, "for/b/foo", mqtt.WriteOptions{}, []byte{2}))
	require.NoError(t, sut.WriteTopic(ctx, "for/c/foo", mqtt.WriteOptions{}, []byte{2}))

	a.Seen.AssertMessage(t, "for/a/foo", []byte{2})
	b.Seen.AssertNoExactMessage(t, "for/b/foo", []byte{2})
	c.Seen.AssertMessage(t, "for/c/foo", []byte{2})

	require.NoError(t, a.WriteTopic(ctx, "for/c/foo", mqtt.WriteOptions{}, []byte("from a")))

	a.RejectWriteWith(fmt.Errorf("dummy"))
	require.EqualError(t, a.WriteTopic(ctx, "for/c/foo", mqtt.WriteOptions{}, []byte("from a: rejected")), "dummy")
	c.Seen.AssertNoExactMessage(t, "for/c/foo", []byte("from a: rejected"))

	a.RejectWriteWith(nil)
	require.NoError(t, a.WriteTopic(ctx, "for/c/foo", mqtt.WriteOptions{}, []byte("from a: after reset reject")))
	c.Seen.AssertMessage(t, "for/c/foo", []byte("from a: after reset reject"))
	a.Sent.AssertMessage(t, "for/c/foo", []byte("from a: after reset reject"))

	c.RejectSubscribeWith(fmt.Errorf("dummy"))
	c.RejectUnsubscribeWith(fmt.Errorf("dummy2"))

	require.EqualError(t, c.Subscribe(ctx, nil, mqtt.Subscription{}), "dummy")
	require.EqualError(t, c.Unsubscribe(ctx, "for/c/foo"), "dummy2")

	c.RejectUnsubscribeWith(nil)
	require.NoError(t, c.Unsubscribe(ctx, "for/c/#"))

	assert.Equal(t, aHandler, 3)
	assert.Equal(t, bHandler, 1)
	assert.Equal(t, cHandler, 4)
}
