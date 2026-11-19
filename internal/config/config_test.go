package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear env variables
	keys := []string{
		"APP_ENV", "APP_PORT", "APP_KEY",
		"ADMIN_USERNAME", "ADMIN_PASSWORD",
		"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD", "DB_SSLMODE",
		"SEARCH_ENGINE_ENDPOINT", "OPENAI_ENDPOINT", "OPENAI_KEY", "OPENAI_MODEL",
	}
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}

	cfg := LoadConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "delcom_blog_secret_key_super_secure_2026_random_string_32chars", cfg.AppKey)
	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.Equal(t, "admin", cfg.AdminPassword)
	assert.Equal(t, "pgsql", cfg.DBConnection)
	assert.Equal(t, "127.0.0.1", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "pabwe_p9_blog", cfg.DBDatabase)
	assert.Equal(t, "postgres", cfg.DBUsername)
	assert.Equal(t, "postgres", cfg.DBPassword)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, "https://delcom.org/api/v1/search", cfg.SearchEngineEndpoint)
	assert.Equal(t, "https://delcom.org/api/v1", cfg.OpenAIEndpoint)
	assert.Equal(t, "key", cfg.OpenAIKey)
	assert.Equal(t, "nama-model", cfg.OpenAIModel)
}

func TestLoadConfig_CustomEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_KEY", "custom_key")
	t.Setenv("ADMIN_USERNAME", "superadmin")
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("DB_CONNECTION", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_DATABASE", "custom_db")
	t.Setenv("DB_USERNAME", "myuser")
	t.Setenv("DB_PASSWORD", "mypass")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("SEARCH_ENGINE_ENDPOINT", "https://search.test")
	t.Setenv("OPENAI_ENDPOINT", "https://ai.test")
	t.Setenv("OPENAI_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "gpt-4")

	cfg := LoadConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, "production", cfg.AppEnv)
	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, "custom_key", cfg.AppKey)
	assert.Equal(t, "superadmin", cfg.AdminUsername)
	assert.Equal(t, "secret", cfg.AdminPassword)
	assert.Equal(t, "postgres", cfg.DBConnection)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "5433", cfg.DBPort)
	assert.Equal(t, "custom_db", cfg.DBDatabase)
	assert.Equal(t, "myuser", cfg.DBUsername)
	assert.Equal(t, "mypass", cfg.DBPassword)
	assert.Equal(t, "require", cfg.DBSSLMode)
	assert.Equal(t, "https://search.test", cfg.SearchEngineEndpoint)
	assert.Equal(t, "https://ai.test", cfg.OpenAIEndpoint)
	assert.Equal(t, "test-key", cfg.OpenAIKey)
	assert.Equal(t, "gpt-4", cfg.OpenAIModel)
}

func TestLoadConfig_WithEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "test.env")
	err := os.WriteFile(envFile, []byte("APP_PORT=7777\n"), 0644)
	assert.NoError(t, err)

	cfg := LoadConfig(envFile)
	assert.NotNil(t, cfg)
	assert.Equal(t, "7777", cfg.AppPort)
}
