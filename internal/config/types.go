package config

import "time"

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type GRPCConfig struct {
	EnableReflection      bool `mapstructure:"enable_reflection"`
	MaxReceiveMessageSize int  `mapstructure:"max_receive_message_size"`
	MaxSendMessageSize    int  `mapstructure:"max_send_message_size"`
}

type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	TokenExpiration     time.Duration `mapstructure:"token_expiration"`
	RefreshTokenEnabled bool          `mapstructure:"refresh_token_enabled"`
}

type AppConfig struct {
	Server ServerConfig `mapstructure:"server"`
	GRPC   GRPCConfig   `mapstructure:"grpc"`
	Auth   AuthConfig   `mapstructure:"auth"`
}
