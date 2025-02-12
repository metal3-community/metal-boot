package main

import (
	"fmt"

	"github.com/appkins-org/go-redfish-uefi/api/redfish"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var redfishSystems map[string]redfish.RedfishSystem

func setupConfigFile() (err error) {
	viper.SetEnvPrefix("bmc")
	viper.BindEnv("tftp_port")
	viper.BindEnv("tftp_root")
	viper.BindEnv("address")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("/app/")
	viper.AddConfigPath("/config/")
	viper.AddConfigPath(".")

	// Load the Config the first time we start the app.
	err = loadConfig()

	// Tell viper to watch the config file.
	viper.WatchConfig()

	// Tell viper what to do when it detects the
	// config file has changed.
	viper.OnConfigChange(func(_ fsnotify.Event) {
		loadConfig()
	})

	return
}

func loadConfig() (err error) {
	// Read the config file into viper and
	// handle (ignore the file) any errors
	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	systems := viper.GetStringMap("systems")
	for key, val := range systems {
		if system, ok := val.(redfish.RedfishSystem); ok {
			redfishSystems[key] = system
		} else {
			// Log an error
			err = fmt.Errorf("Invalid system configuration")
		}
	}
	return
}
