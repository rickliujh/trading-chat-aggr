package main

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type Config struct {
	DBURI         string   `mapstructure:"dburi"`
	Addr          string   `mapstructure:"addr"`
	Symbols       []string `mapstructure:"symbols"`
	LogLevel      int      `mapstructure:"log_level"`
	EnablePush    bool     `mapstructure:"enable_push"`
	EnablePersist bool     `mapstructure:"enable_persist"`
}

func setDefault() {
	viper.SetDefault("ADDR", "localhost:8080")
	viper.SetDefault("DBURI", "postgres://username:password@localhost:5432/database_name")
	viper.SetDefault("SYMBOLS", "ETHBTC,BNBBTC")
	viper.SetDefault("LOG_LEVEL", 0)
	viper.SetDefault("ENABLE_PUSH", true)
	viper.SetDefault("ENABLE_PERSIST", false)
}

func loadConfig() (Config, error) {
	setDefault()
	var conf Config
	viper.AutomaticEnv()
	err := viper.Unmarshal(&conf, func(dc *mapstructure.DecoderConfig) {
		dc.ErrorUnset = true
	})
	return conf, err
}
