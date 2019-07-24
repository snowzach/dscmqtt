package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"net/http"
	_ "net/http/pprof" // Import for pprof

	config "github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.SugaredLogger
)

func main() {

	configFile := flag.String("config", "", "Config file")

	// Sets up the config file, environment etc
	config.SetTypeByDefaultValue(true)                      // If a default value is []string{"a"} an environment variable of "a b" will end up []string{"a","b"}
	config.AutomaticEnv()                                   // Automatically use environment variables where available
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // Environement variables use underscores instead of periods

	// If a config file is found, read it in.
	if *configFile != "" {
		config.SetConfigFile(*configFile)
		err := config.ReadInConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not read config file: %s ERROR: %s\n", configFile, err.Error())
			os.Exit(1)
		}

	}

	logConfig := zap.NewProductionConfig()

	// Log Level
	var logLevel zapcore.Level
	if err := logLevel.Set(config.GetString("logger.level")); err != nil {
		zap.S().Fatalw("Could not determine logger.level", "error", err)
	}
	logConfig.Level.SetLevel(logLevel)

	// Settings
	logConfig.Encoding = config.GetString("logger.encoding")
	logConfig.Development = config.GetBool("logger.dev_mode")
	logConfig.DisableCaller = config.GetBool("logger.disable_caller")
	logConfig.DisableStacktrace = config.GetBool("logger.disable_stacktrace")

	// Enable Color
	if config.GetBool("logger.color") {
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Use sane timestamp when logging to console
	if logConfig.Encoding == "console" {
		logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	globalLogger, _ := logConfig.Build()
	zap.ReplaceGlobals(globalLogger)
	logger = globalLogger.Sugar().With("package", "cmd")

	// Profliter can explicitly listen on address/port
	if config.GetBool("profiler.enabled") {
		hostPort := net.JoinHostPort(config.GetString("profiler.host"), config.GetString("profiler.port"))
		go http.ListenAndServe(hostPort, nil)
		logger.Infof("Profiler enabled on http://%s", hostPort)
	}

	// Init the panel
	panel, err := NewDSCPanel()
	if err != nil {
		logger.Fatalw("Could not initialize DSCPanel", "error", err)
	}

	mqtt, err := NewMQTTClient()
	if err != nil {
		logger.Fatalw("Could not initialize MQTTClient", "error", err)
	}

	// Get the full status of everything at regular intervals
	if config.GetDuration("dsc.full_update_interval") != 0 {
	        go func() {
			for {
	                        logger.Info("Requesting full update")
				panel.FullUpdate()
				time.Sleep(config.GetDuration("dsc.full_update_interval"))
			}
		}()
	}

	topic := config.GetString("mqtt.topic")

	// Main Loop
	for {
		m := panel.GetMessage(true)
		if m.Err != nil {
			logger.Errorw("DSCError", "error", m.Err)
			continue
		}
		switch m.Type {
		case DSC_TYPE_ZONE:
			switch m.State {
			case DSC_STATE_OPEN:
				mqtt.Publish(fmt.Sprintf("%s/%s", topic, m.Id), MQTT_STATE_ON)
			case DSC_STATE_CLOSED:
				mqtt.Publish(fmt.Sprintf("%s/%s", topic, m.Id), MQTT_STATE_OFF)
			}
			logger.Infow("Zone State", "state", m.State, "id", m.Id)
		case DSC_TYPE_VERSION:
                        logger.Infof("Version command detected...")
			location, err := time.LoadLocation(config.GetString("dsc.time_zone"))
			if err != nil {
				logger.Fatalf("Could not load timezone data: %v", err)
			}
			now := time.Now().In(location)
                        logger.Infof("Updating time to %s", now)
			go panel.TimeUpdate(now)
		default:
			logger.Infow("Panel Message", "type", m.Type, "state", m.State, "id", m.Id, "error", m.Err)
		}

	}

}
