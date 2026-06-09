package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAllowsEnvironmentOverrides(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configBody := []byte(`
server:
  host: "127.0.0.1"
  port: "8080"
database:
  driver: "sqlite"
  path: "./data/scenic_guide.db"
logging:
  level: "info"
  output: "console"
ai:
  api_key: ""
  model: "deepseek-chat"
  base_url: "https://api.deepseek.com/v1"
embedding:
  api_key: ""
  model: "text-embedding-v2"
  base_url: "https://dashscope.aliyuncs.com/api/v1"
speech:
  api_key: ""
  region: ""
security:
  jwt_secret: "file-secret"
  token_expire_hours: 24
`)
	if err := os.WriteFile(configPath, configBody, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("SCENIC_GUIDE_SERVER_PORT", "9090")
	t.Setenv("SCENIC_GUIDE_SECURITY_TOKEN_EXPIRE_HOURS", "4")
	t.Setenv("SCENIC_GUIDE_SECURITY_ALLOWED_ORIGINS", "http://localhost:5173, https://example.com ")
	t.Setenv("SCENIC_GUIDE_AI_API_KEY", "env-api-key")

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Fatalf("expected env server port, got %q", cfg.Server.Port)
	}
	if cfg.Security.TokenExpireHours != 4 {
		t.Fatalf("expected env token expiry, got %d", cfg.Security.TokenExpireHours)
	}
	if cfg.AI.APIKey != "env-api-key" {
		t.Fatalf("expected env AI key override")
	}
	if len(cfg.Security.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %#v", cfg.Security.AllowedOrigins)
	}
	if cfg.Security.AllowedOrigins[0] != "http://localhost:5173" || cfg.Security.AllowedOrigins[1] != "https://example.com" {
		t.Fatalf("unexpected allowed origins: %#v", cfg.Security.AllowedOrigins)
	}
}
