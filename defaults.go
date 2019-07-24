package main

import (
	config "github.com/spf13/viper"
)

func init() {

	// Logger Defaults
	config.SetDefault("logger.level", "info")
	config.SetDefault("logger.encoding", "console")
	config.SetDefault("logger.color", true)
	config.SetDefault("logger.dev_mode", true)
	config.SetDefault("logger.disable_caller", false)
	config.SetDefault("logger.disable_stacktrace", true)

	// Profiler config
	config.SetDefault("profiler.enabled", false)
	config.SetDefault("profiler.host", "")
	config.SetDefault("profiler.port", "6060")

	// DSC Settings
	config.SetDefault("dsc.port", "/dev/alarmsystem")
	config.SetDefault("dsc.baud", 9600)
	config.SetDefault("dsc.full_update_interval", "60m")
        config.SetDefault("dsc.time_zone", "Local")

	// MQTT Settings
	config.SetDefault("mqtt.host", "localhost")
	config.SetDefault("mqtt.port", 1883)
	config.SetDefault("mqtt.username", "mqtt")
	config.SetDefault("mqtt.password", "mqtt")
	config.SetDefault("mqtt.client_id", "dscmqtt")
	config.SetDefault("mqtt.topic", "dsc/zone")

}
