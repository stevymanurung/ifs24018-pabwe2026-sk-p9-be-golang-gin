package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv               string
	AppPort              string
	AppKey               string
	AdminUsername        string
	AdminPassword        string
	DBConnection         string
	DBHost               string
	DBPort               string
	DBDatabase           string
	DBUsername           string
	DBPassword           string
	DBSSLMode            string
	SearchEngineEndpoint string
	OpenAIEndpoint       string
	OpenAIKey            string
	OpenAIModel          string
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func LoadConfig(envFiles ...string) *Config {
	if len(envFiles) > 0 {
		_ = godotenv.Load(envFiles...)
	} else {
		_ = godotenv.Load()
	}

	return &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		AppPort:              getEnv("PORT", getEnv("APP_PORT", "8081")),
		AppKey:               getEnv("APP_KEY", "delcom_blog_secret_key_super_secure_2026_random_string_32chars"),
		AdminUsername:        getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:        getEnv("ADMIN_PASSWORD", "admin"),
		DBConnection:         getEnv("DB_CONNECTION", "pgsql"),
		DBHost:               getEnv("DB_HOST", "127.0.0.1"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBDatabase:           getEnv("DB_DATABASE", "pabwe_p9_blog"),
		DBUsername:           getEnv("DB_USERNAME", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", "postgres"),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		SearchEngineEndpoint: getEnv("SEARCH_ENGINE_ENDPOINT", "https://delcom.org/api/v1/search"),
		OpenAIEndpoint:       getEnv("OPENAI_ENDPOINT", "https://delcom.org/api/v1"),
		OpenAIKey:            getEnv("OPENAI_KEY", "key"),
		OpenAIModel:          getEnv("OPENAI_MODEL", "nama-model"),
	}
}
