package server

import (
	"fmt"
	"os"

	"github.com/elskow/chef-infra/internal/config"
	"github.com/spf13/viper"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTesting     = "testing"
)

func LoadConfig() (*config.AppConfig, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = EnvDevelopment
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath("./config/server")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config config.AppConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Load environment-specific configurations
	if envSettings := v.GetStringMap(fmt.Sprintf("grpc.%s", env)); len(envSettings) > 0 {
		if err := v.UnmarshalKey(fmt.Sprintf("grpc.%s", env), &config.GRPC); err != nil {
			return nil, fmt.Errorf("error unmarshaling env config: %w", err)
		}
	}

	return &config, nil
}
