package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	AI        AIConfig        `mapstructure:"ai"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Speech    SpeechConfig    `mapstructure:"speech"`
	Security  SecurityConfig  `mapstructure:"security"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Driver                 string `mapstructure:"driver"`
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	Name                   string `mapstructure:"name"`
	User                   string `mapstructure:"user"`
	Password               string `mapstructure:"password"`
	Path                   string `mapstructure:"path"`
	MaxOpenConns           int    `mapstructure:"max_open_conns"`
	MaxIdleConns           int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `mapstructure:"conn_max_lifetime_minutes"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Output string `mapstructure:"output"`
}

type AIConfig struct {
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	BaseURL string `mapstructure:"base_url"`
}

type EmbeddingConfig struct {
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	BaseURL string `mapstructure:"base_url"`
}

type SpeechConfig struct {
	APIKey string `mapstructure:"api_key"`
	Region string `mapstructure:"region"`
}

type SecurityConfig struct {
	JWTSecret        string `mapstructure:"jwt_secret"`
	TokenExpireHours int    `mapstructure:"token_expire_hours"`
}

func LoadConfig(path string) (*Config, error) {
	viper.Reset()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.SetEnvPrefix("SCENIC_GUIDE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	bindEnvKeys()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func bindEnvKeys() {
	keys := []string{
		"server.host",
		"server.port",
		"database.driver",
		"database.host",
		"database.port",
		"database.name",
		"database.user",
		"database.password",
		"database.path",
		"database.max_open_conns",
		"database.max_idle_conns",
		"database.conn_max_lifetime_minutes",
		"logging.level",
		"logging.output",
		"ai.api_key",
		"ai.model",
		"ai.base_url",
		"embedding.api_key",
		"embedding.model",
		"embedding.base_url",
		"speech.api_key",
		"speech.region",
		"security.jwt_secret",
		"security.token_expire_hours",
	}
	for _, key := range keys {
		_ = viper.BindEnv(key)
	}
}
