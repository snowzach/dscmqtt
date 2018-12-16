package main

import (
	"crypto/tls"
	"fmt"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	config "github.com/spf13/viper"
)

const (
	MQTT_STATE_ON  = "ON"
	MQTT_STATE_OFF = "OFF"
)

type MQTTClient struct {
	client MQTT.Client
}

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	logger.Info("MQTT Message", "topic", message.Topic(), "payload", message.Payload())
}

func NewMQTTClient() (*MQTTClient, error) {

	connOpts := MQTT.NewClientOptions().
		AddBroker("tcp://" + config.GetString("mqtt.host") + ":" + config.GetString("mqtt.port")).
		SetClientID(config.GetString("mqtt.client_id")).
		SetCleanSession(true)

	if config.GetString("mqtt.username") != "" {
		connOpts.SetUsername(config.GetString("mqtt.username"))
		if config.GetString("mqtt.password") != "" {
			connOpts.SetPassword(config.GetString("mqtt.password"))
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	connOpts.OnConnect = func(c MQTT.Client) {
		if token := c.Subscribe(config.GetString("mqtt.topic"), byte(0), onMessageReceived); token.Wait() && token.Error() != nil {
			logger.Fatalf("Connect Error: %s", token.Error())
		}
	}

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("Connect error: %s", token.Error())
	} else {
		logger.Debug("Connected to MQTT")
	}

	return &MQTTClient{
		client: client,
	}, nil

}

func (c *MQTTClient) Publish(topic string, message string) {
	if !c.client.Publish(topic, byte(0), true, message).WaitTimeout(5 * time.Second) {
		logger.Error("Could not Publish Message")
	}

}
