package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	Env, Port, AppKey                                      string
	AdminUsername, AdminPassword                           string
	DBHost, DBPort, DBName, DBUser, DBPassword, DBSSLMode  string
	SearchEndpoint, OpenAIEndpoint, OpenAIKey, OpenAIModel string
}

func Load() Config {
	_ = godotenv.Load()
	get := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return Config{
		Env: get("APP_ENV", "development"), Port: get("APP_PORT", "8081"),
		AppKey:        get("APP_KEY", "local-development-secret-change-this-32chars"),
		AdminUsername: get("ADMIN_USERNAME", "admin"), AdminPassword: get("ADMIN_PASSWORD", "admin"),
		DBHost: get("DB_HOST", "127.0.0.1"), DBPort: get("DB_PORT", "5432"),
		DBName: get("DB_DATABASE", "pabwe_p9_blog"), DBUser: get("DB_USERNAME", "postgres"),
		DBPassword: get("DB_PASSWORD", "postgres"), DBSSLMode: get("DB_SSLMODE", "disable"),
		SearchEndpoint: get("SEARCH_ENGINE_ENDPOINT", "https://delcom.org/api/v1/search"),
		OpenAIEndpoint: get("OPENAI_ENDPOINT", "https://delcom.org/api/v1"),
		OpenAIKey:      get("OPENAI_KEY", ""), OpenAIModel: get("OPENAI_MODEL", ""),
	}
}
