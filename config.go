package main

import (
	"github.com/appkins-org/go-redfish-uefi/api/redfish"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var redfishSystems map[string]redfish.RedfishSystem

type UnifiConfig struct {
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	Site     string `yaml:"site" mapstructure:"site"`
	Device   string `yaml:"device" mapstructure:"device"`
}

type Config struct {
	Address string                           `yaml:"address" mapstructure:"address"`
	Port    int                              `yaml:"port" mapstructure:"port"`
	Unifi   UnifiConfig                      `yaml:"unifi" mapstructure:"unifi"`
	Systems map[string]redfish.RedfishSystem `yaml:"systems" mapstructure:"systems"`
}

func NewConfig() (conf *Config, err error) {
	conf = &Config{}

	viper.AddConfigPath("/app/")
	viper.AddConfigPath("/config/")
	viper.AddConfigPath(".")

	viper.SetConfigName("redfish")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	// Load the Config the first time we start the app.
	err = loadConfig(conf)
	if err != nil {
		return
	}

	// Tell viper to watch the config file.
	viper.WatchConfig()

	// Tell viper what to do when it detects the
	// config file has changed.
	viper.OnConfigChange(func(_ fsnotify.Event) {
		_ = loadConfig(conf)
	})

	return
}

func loadConfig(conf *Config) (err error) {
	// read the config file into viper and
	// handle (ignore the file) any errors
	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(conf)
	if err != nil {
		return
	}

	return
}
