package config

import (
	"github.com/spf13/viper"
)

func MustLoad() {
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic("error read cfg file")
	}
}
