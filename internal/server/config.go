package server

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	GRPC   GRPCConfig   `mapstructure:"grpc"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type GRPCConfig struct {
	EnableReflection      bool `mapstructure:"enable_reflection"`
	MaxReceiveMessageSize int  `mapstructure:"max_receive_message_size"`
	MaxSendMessageSize    int  `mapstructure:"max_send_message_size"`
}

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTesting     = "testing"
)

func LoadConfig() (*Config, error) {
	// Get environment from ENV variable, default to development
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = EnvDevelopment
	}

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("./config/server")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Get base config
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling base config: %w", err)
	}

	// Override with environment-specific settings
	envSettings := viper.GetStringMap(fmt.Sprintf("grpc.%s", env))
	if len(envSettings) > 0 {
		if err := viper.UnmarshalKey(fmt.Sprintf("grpc.%s", env), &config.GRPC); err != nil {
			return nil, fmt.Errorf("error unmarshaling env config: %w", err)
		}
	}

	return &config, nil
}
