package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logging  LoggingConfig
	AI       AIConfig
	Speech   SpeechConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	Path     string
}

type LoggingConfig struct {
	Level  string
	Output string
}

type AIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type SpeechConfig struct {
	APIKey string
	Region string
}

type SecurityConfig struct {
	JWTSecret       string
	TokenExpireHours int
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
