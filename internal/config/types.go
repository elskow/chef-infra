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
	JWTSecret            string        `mapstructure:"jwt_secret"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration"`
	RefreshTokenEnabled  bool          `mapstructure:"refresh_token_enabled"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Database DatabaseConfig `mapstructure:"database"`
}
