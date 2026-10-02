package main

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
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

	topicPrefix := "hqtt/example"

	// A single availability topic for all devices / entities produced by this integration
	availability := mqtt.NewAbsoluteValueWithOptions[hass.Availability](
		mqtt.JoinTopic(topicPrefix, "availability"),
		hass.AvailabilityMarshaler,
		mqtt.WriteOptions{
			Retain: true,
			QoS:    mqtt.QOSAtLeastOnce,
		},
	)

	w, sm, hassAvailability, disconnect, err := configureMQTT(ctx, brokerURL, availability)
	if err != nil {
		panic(err)
	}

	log := hqttlog.ForComponent("example")
	log.Info("Starting Up")

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := disconnect(shutdownCtx); err != nil {
			log.With(hqttlog.Error(err)).Error("Failed to disconnect from mqtt")
		}

		log.Info("MQTT Disconnected")
	}()

	// Wait for Home Assistant to be available
	if err = mqtt.Error(hassAvailability.Await(ctx, mqtt.DesiredValue(hass.Available))); err != nil {
		panic(err)
	}

	log.Info("Home Assistant is now available")

	// Setup Discovery
	log.Info("Setting up device")
	l := hqtt.Component[*platform.Light]{
		UniqueID:    "example.foo",
		Name:        "Foo",
		TopicPrefix: mqtt.JoinTopic(topicPrefix, "foo"),

		DefaultEntityID: "light.foo",
		Icon:            "mdi:floor-lamp",

		Availability: availability,

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
		Icon:            "mdi:motion-sensor",

		Platform: platform.NewBinarySensor(
			mqtt.NewValueWithOptions[hass.PowerState]("state", hass.PowerStateMarshaler, mqtt.WriteOptions{Retain: true}),
			platform.NewSensorAttributeValue[map[string]any]("attributes", nil),
		),

		Availability: availability,
	}

	d := &hqtt.Device{
		Name:        "Example Device",
		Identifiers: []string{"hqtt/example/fake_light"},
	}

	log.Info("Watching Command Topics")
	l.Platform.Command.Watch(func(s hass.PowerState) {
		log.With(slog.Any("state", s)).Info("Home Assistant sent light command")
		if err = mqtt.Error(l.Platform.State.Write(ctx, w, l.TopicPrefix, s)); err != nil {
			log.With(hqttlog.Error(err)).Error("Failed to update light state after receiving state command")
		}
	})
	l.Platform.ColorModeCommand.Watch(func(m hass.ColorMode) {
		log.With(slog.Any("mode", m)).Info("Home Assistant set color mode")
		if err = mqtt.Error(l.Platform.ColorMode.Write(ctx, w, l.TopicPrefix, m)); err != nil {
			log.With(hqttlog.Error(err)).Error("Failed to update color mode after receiving mode command")
		}
	})
	l.Platform.BrightnessCommand.Watch(func(u uint) {
		log.With(slog.Any("brightness", u)).Info("Home Assistant set brightness command")
		if err = mqtt.Error(l.Platform.Brightness.Write(ctx, w, l.TopicPrefix, u)); err != nil {
			log.With(hqttlog.Error(err)).Error("Failed to update brightness after receiving brightness command")
		}
	})
	l.Platform.ColorTemperatureCommand.Watch(func(u uint) {
		log.With(slog.Any("color-temp", u)).Info("Home Assistant set color temperature")

		if err = errors.Join(
			mqtt.Error(l.Platform.ColorTemperature.Write(ctx, w, l.TopicPrefix, u)),
			mqtt.Error(l.Platform.ColorMode.Write(ctx, w, l.TopicPrefix, hass.ColorModeTemperature)),
		); err != nil {
			log.With(hqttlog.Error(err)).Error("Failed to update color temperature and/or mode after receiving color temperature command")
		}
	})
	l.Platform.RGBCommand.Watch(func(rgb platform.RGB) {
		log.With(slog.Any("rgb", rgb)).Info("Home Assistant set color")

		if err = errors.Join(
			mqtt.Error(l.Platform.RGB.Write(ctx, w, l.TopicPrefix, rgb)),
			mqtt.Error(l.Platform.ColorMode.Write(ctx, w, l.TopicPrefix, hass.ColorModeRGB)),
		); err != nil {
			log.With(hqttlog.Error(err)).Error("Failed to update rgb and/or color mode after receiving rgb command")
		}
	})

	go func() {
		log.Info("Starting fake presence sensor loop")

		for {
			select {
			case <-time.After(time.Duration(rand.IntN(10)+1) * time.Second):
				last, ok := s.Platform.Sensor.State.Get()

				next := hass.PowerStateUnknown
				if !ok {
					next = hass.PowerStateOn
				} else if last == hass.PowerStateOn {
					next = hass.PowerStateOff
				} else {
					next = hass.PowerStateOn
				}

				log.With(slog.Any("state", next)).Debug("Updating fake presence sensor state")
				if err := mqtt.Error(s.Platform.Sensor.State.Write(ctx, w, s.TopicPrefix, next)); err != nil {
					log.With(hqttlog.Error(err)).Error("Failed to update fake presence sensor state")
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	components := map[string]discovery.ComponentMarshaler{
		l.UniqueID: &l,
		s.UniqueID: &s,
	}

	// TODO: Plumb through retain
	rediscover := func() error {
		log.Info("Re-sending discovery info")
		return d.Configure(ctx, w, mqtt.WriteOptions{Retain: true}, discovery.DefaultPrefix, components)
	}

	republish := func() error {
		log.Info("Republishing state")
		return errors.Join(
			mqtt.Error(l.Platform.State.Write(ctx, w, l.TopicPrefix, hass.PowerStateOff)),
			mqtt.Error(s.Platform.State.Write(ctx, w, s.TopicPrefix, hass.PowerStateOff)),
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

	if err = errors.Join(
		rediscover(),
		republish(),
	); err != nil {
		panic(err)
	}

	<-ctx.Done()
	log.Info("Goodbye!")
}
