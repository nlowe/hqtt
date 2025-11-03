package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"

	"github.com/nlowe/hqtt/discovery"
	"github.com/nlowe/hqtt/hass"
	hqttlog "github.com/nlowe/hqtt/log"
	"github.com/nlowe/hqtt/mqtt"
	adapter "github.com/nlowe/hqtt/mqtt/adapter/autopaho"
)

type disconnectFunc func(context.Context) error

func configureMQTT(ctx context.Context, brokerURL *url.URL) (mqtt.Writer, mqtt.Subscriber, *mqtt.RemoteValue[hass.Availability], disconnectFunc, error) {
	log := hqttlog.ForComponent("mqtt")

	mqttConfig := autopaho.ClientConfig{
		ServerUrls: []*url.URL{brokerURL},
		KeepAlive:  20,

		// SessionExpiryInterval - Seconds that a session will survive after disconnection. It is important to set this
		// because otherwise, any queued messages will be lost if the connection drops and the server will not queue
		// messages while it is down. The specific setting will depend upon your needs (60 = 1 minute, 3600 = 1 hour,
		// 86400 = one day, 0xFFFFFFFE = 136 years, 0xFFFFFFFF = don't expire)
		SessionExpiryInterval: 60,

		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Info("mqtt connected")
		},
		OnConnectError: func(err error) {
			slog.With(hqttlog.Error(err)).Error("mqtt connection error")
		},

		ClientConfig: paho.ClientConfig{
			ClientID: "hqtt:example:fake_light",
			OnClientError: func(err error) {
				log.With(hqttlog.Error(err)).Error("mqtt client error")
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				log := log.With(slog.Int("reason", int(d.ReasonCode)))

				if d.Properties != nil {
					log = log.With(
						slog.Group(
							"properties",
							slog.String("reference", d.Properties.ServerReference),
							slog.String("reason", d.Properties.ReasonString),
							slog.Any("user", d.Properties.User),
						),
					)
				}

				log.Warn("Disconnected from server")
			},
		},
	}

	log.With(slog.String("broker", brokerURL.String())).Info("Connecting to mqtt")
	w, s, disconnect, err := adapter.DialMQTT(ctx, mqttConfig)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("mqtt: connect: %w", err)
	}

	log.With(slog.String("broker", brokerURL.String())).Info("Connected to mqtt")

	hassAvailability := discovery.HomeAssistantAvailability(discovery.DefaultPrefix)
	if err = s.Subscribe(ctx, hassAvailability, mqtt.Subscription{Topic: hassAvailability.FullyQualifiedTopic("")}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("subscribe to home assistant status: %w", err)
	}

	return w, s, hassAvailability, disconnect, err
}
