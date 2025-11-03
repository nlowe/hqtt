package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/nlowe/hqtt"
	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/hass"
	hqttlog "github.com/nlowe/hqtt/log"
	"github.com/nlowe/hqtt/mqtt"
	"github.com/nlowe/hqtt/platform"
)

func main() {
	hqttlog.To(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	brokerURL, err := url.Parse("mqtt://0.0.0.0:1883")
	if err != nil {
		panic(err)
	}

	w, sm, hassAvailability, disconnect, err := configureMQTT(ctx, brokerURL)
	if err != nil {
		panic(err)
	}

	log := hqttlog.ForComponent("example")
	log.Info("Starting Up")

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		log.Info("Disconnecting from mqtt")
		if err := disconnect(shutdownCtx); err != nil {
			log.With(hqttlog.Error(err)).Error("Failed to disconnect from mqtt")
		}
	}()

	// Wait for Home Assistant to be available
	_, err = hassAvailability.Await(ctx, mqtt.DesiredValue(hass.Available))

	if err != nil {
		panic(err)
	}

	log.Info("Home Assistant is now available")

	topicPrefix := "hqtt/example"

	// Setup Discovery
	log.Info("Setting up device")
	l := hqtt.Component[*platform.Light]{
		UniqueID:    "example.foo",
		Name:        "Foo",
		TopicPrefix: mqtt.JoinTopic(topicPrefix, "foo"),

		DefaultEntityID: "light.foo",
		Icon:            "mdi:light",

		Availability: mqtt.NewValueWithOptions[hass.Availability]("available", hass.AvailabilityMarshaler, mqtt.WriteOptions{Retain: true}),

		Platform: &platform.Light{
			OnCommandType: platform.LightOnCommandTypeLast,

			State:   mqtt.NewValueWithOptions[hass.PowerState]("state", hass.PowerStateMarshaler, mqtt.WriteOptions{Retain: true}),
			Command: mqtt.NewRemoteValue[hass.PowerState]("command", hass.PowerStateUnmarshaler),

			ColorMode:        mqtt.NewValue[hass.ColorMode](mqtt.JoinTopic("color", "mode"), hass.ColorModeMarshaler),
			ColorModeCommand: mqtt.NewRemoteValue[hass.ColorMode](mqtt.JoinTopic("mode", "set"), hass.ColorModeUnmarshaler),
			SupportedColorModes: []hass.ColorMode{
				hass.ColorModeTemperature,
				hass.ColorModeRGB,
			},

			Brightness:        mqtt.NewValue[uint]("brightness", mqtt.UintMarshaler),
			BrightnessCommand: mqtt.NewRemoteValue[uint](mqtt.JoinTopic("brightness", "set"), mqtt.UintUnmarshaler),
			BrightnessScale:   100,

			ColorTemperature:         mqtt.NewValue[uint](mqtt.JoinTopic("color", "temperature"), mqtt.UintMarshaler),
			ColorTemperatureCommand:  mqtt.NewRemoteValue[uint](mqtt.JoinTopic("color", "temperature", "set"), mqtt.UintUnmarshaler),
			ColorTemperatureInKelvin: true,
			MaxKelvin:                9000,
			MinKelvin:                2000,

			RGB:        mqtt.NewValue[platform.RGB](mqtt.JoinTopic("color", "rgb"), platform.RGBMarshaler),
			RGBCommand: mqtt.NewRemoteValue[platform.RGB](mqtt.JoinTopic("color", "rgb", "set"), platform.RGBUnmarshaler),
		},

		WriteOptions: mqtt.WriteOptions{
			Retain: true,
		},
	}
	if err = l.Subscribe(ctx, sm); err != nil {
		panic(err)
	}

	s := hqtt.Component[*platform.BinarySensor[map[string]any]]{
		UniqueID:    "example.foo.pir",
		Name:        "Foo Presence",
		TopicPrefix: mqtt.JoinTopic(topicPrefix, "foo_pir"),

		DefaultEntityID: "binary_sensor.foo_pir",
		Icon:            "mdi:light",

		Platform: platform.NewBinarySensor(
			mqtt.NewValueWithOptions[hass.PowerState]("state", hass.PowerStateMarshaler, mqtt.WriteOptions{Retain: true}),
			platform.NewSensorAttributeValue[map[string]any]("attributes", nil),
		),

		Availability: mqtt.NewValueWithOptions[hass.Availability]("available", hass.AvailabilityMarshaler, mqtt.WriteOptions{Retain: true}),
	}

	d := &hqtt.Device{
		Name:        "Example Device",
		Identifiers: []string{"hqtt/example/fake_light"},
	}

	log.Info("Watching Command Topics")
	l.Platform.Command.Watch(func(s hass.PowerState) {
		log.With(slog.Any("state", s)).Info("Home Assistant sent light command")
		_, _ = l.Platform.State.Write(ctx, w, l.TopicPrefix, s)
	})
	l.Platform.ColorModeCommand.Watch(func(m hass.ColorMode) {
		log.With(slog.Any("mode", m)).Info("Home Assistant set color mode")
		_, _ = l.Platform.ColorMode.Write(ctx, w, l.TopicPrefix, m)
	})
	l.Platform.BrightnessCommand.Watch(func(u uint) {
		log.With(slog.Any("brightness", u)).Info("Home Assistant set brightness command")
		_, _ = l.Platform.Brightness.Write(ctx, w, l.TopicPrefix, u)
	})
	l.Platform.ColorTemperatureCommand.Watch(func(u uint) {
		log.With(slog.Any("color-temp", u)).Info("Home Assistant set color temperature")
		_, _ = l.Platform.ColorTemperature.Write(ctx, w, l.TopicPrefix, u)
		_, _ = l.Platform.ColorMode.Write(ctx, w, l.TopicPrefix, hass.ColorModeTemperature)
	})
	l.Platform.RGBCommand.Watch(func(rgb platform.RGB) {
		log.With(slog.Any("rgb", rgb)).Info("Home Assistant set color")
		_, _ = l.Platform.RGB.Write(ctx, w, l.TopicPrefix, rgb)
		_, _ = l.Platform.ColorMode.Write(ctx, w, l.TopicPrefix, hass.ColorModeRGB)
	})

	components := map[string]json.MarshalerTo{
		l.UniqueID: &l,
		s.UniqueID: &s,
	}

	// TODO: Plumb through retain
	rediscover := func() error {
		log.Info("Re-sending discovery info")
		return d.Configure(ctx, w, discovery.DefaultPrefix, components)
	}

	republish := func() error {
		log.Info("Republishing state/availability")
		return errors.Join(
			mqtt.Error(l.Platform.State.Write(ctx, w, l.TopicPrefix, hass.PowerStateOff)),
			mqtt.Error(l.Availability.Write(ctx, w, l.TopicPrefix, hass.Available)),
			mqtt.Error(s.Platform.State.Write(ctx, w, s.TopicPrefix, hass.PowerStateOff)),
			mqtt.Error(s.Availability.Write(ctx, w, s.TopicPrefix, hass.Available)),
		)
	}

	log.Info("Watching Home Assistant state")
	hassAvailability.Watch(func(availability hass.Availability) {
		log.With("availability", availability).Info("Home Assistant state changed")
		if availability == hass.Available {
			if err := errors.Join(rediscover(), republish()); err != nil {
				panic(err)
			}
		}
	})

	if err = rediscover(); err != nil {
		panic(err)
	}

	if err = republish(); err != nil {
		panic(err)
	}

	<-ctx.Done()
	log.Info("Goodbye!")
}
