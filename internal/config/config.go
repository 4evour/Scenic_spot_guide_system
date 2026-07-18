package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	AI         AIConfig         `mapstructure:"ai"`
	Embedding  EmbeddingConfig  `mapstructure:"embedding"`
	Speech     SpeechConfig     `mapstructure:"speech"`
	TTS        TTSConfig        `mapstructure:"tts"`
	Multimodal MultimodalConfig `mapstructure:"multimodal"`
	Security   SecurityConfig   `mapstructure:"security"`
	Redis      RedisConfig      `mapstructure:"redis"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
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

// TTSConfig holds Text-to-Speech configuration.
type TTSConfig struct {
	Provider string `mapstructure:"provider"` // "edge" (default, currently the only supported provider)
	Voice    string `mapstructure:"voice"`    // e.g. "female_xiaoxiao"
	Rate     string `mapstructure:"rate"`     // e.g. "+0%", "-10%", "+20%"
	Timeout  int    `mapstructure:"timeout"`  // seconds, default 30
}

// MultimodalConfig configures the optional Qwen Omni compatible endpoint.
type MultimodalConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Provider       string `mapstructure:"provider"`
	Model          string `mapstructure:"model"`
	BaseURL        string `mapstructure:"base_url"`
	APIKey         string `mapstructure:"api_key"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

func (c MultimodalConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Provider) == "" {
		return fmt.Errorf("multimodal.provider is required when multimodal is enabled")
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("multimodal.model is required when multimodal is enabled")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("multimodal.base_url is required when multimodal is enabled")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("multimodal.api_key is required when multimodal is enabled")
	}
	if c.TimeoutSeconds <= 0 || c.TimeoutSeconds > 300 {
		return fmt.Errorf("multimodal.timeout_seconds must be between 1 and 300")
	}
	return nil
}

type SecurityConfig struct {
	JWTSecret        string   `mapstructure:"jwt_secret"`
	TokenExpireHours int      `mapstructure:"token_expire_hours"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
}

func LoadConfig(path string) (*Config, error) {
	viper.Reset()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.SetEnvPrefix("SCENIC_GUIDE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetDefault("multimodal.enabled", false)
	viper.SetDefault("multimodal.provider", "qwen")
	viper.SetDefault("multimodal.model", "qwen3.5-omni-plus")
	viper.SetDefault("multimodal.base_url", "https://dashscope.aliyuncs.com/compatible-mode/v1")
	viper.SetDefault("multimodal.timeout_seconds", 60)
	bindEnvKeys()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	if origins := os.Getenv("SCENIC_GUIDE_SECURITY_ALLOWED_ORIGINS"); origins != "" {
		config.Security.AllowedOrigins = splitCSV(origins)
	}

	// JWT is mandatory; AI credentials are optional because local RAG is the supported fallback.
	if config.Security.JWTSecret == "" {
		return nil, fmt.Errorf("security.jwt_secret is not configured, set SCENIC_GUIDE_SECURITY_JWT_SECRET environment variable")
	}
	if err := config.Multimodal.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
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
		"multimodal.enabled",
		"multimodal.provider",
		"multimodal.model",
		"multimodal.base_url",
		"multimodal.api_key",
		"multimodal.timeout_seconds",
		"security.jwt_secret",
		"security.token_expire_hours",
		"security.allowed_origins",
		"redis.addr",
		"redis.password",
		"redis.db",
	}
	for _, key := range keys {
		_ = viper.BindEnv(key)
	}
}
