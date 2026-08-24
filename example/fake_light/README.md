# hqtt fake light example

To setup:

1. `docker-compose up -d`
2. Go to http://localhost:8123
3. Finish initial setup
4. *[Manually](https://my.home-assistant.io/redirect/config_flow_start?domain=mqtt) configure the MQTT Integration*. Yes really. [Home Assistant removed the ability for me to do it for you with a default yaml config](https://github.com/home-assistant/architecture/blob/master/adr/0010-integration-configuration.md#decision).
    1. Settings > Devices & services > Add Integration: `MQTT`
       * Broker: `mqtt`
       * Port: `1883`
       * MQTT Protocol: `5`
    2. Click the settings gear > Configure MQTT options
        * Enable discovery: `Enabled`
          * Discovery prefix: `homeassistant`
        * Enable birth message: `Enabled`
          * Birth message topic: `homeassistant/status`
          * Birth message payload: `online`
          * Birth message retain: `Enabled`
        * Enable will message: `Enabled`
          * Will message topic: `homeassistant/status`
          * Will message payload: `offline`
          * Will message retain: `Enabled`
