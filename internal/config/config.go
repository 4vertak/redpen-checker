package config

import "os"

type Config struct {
	Port           string
	DatabaseURL    string
	GigaChatAPIKey string
	DeepSeekAPIKey string
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/redpen?sslmode=disable"),
		GigaChatAPIKey: getEnv("GIGACHAT_API_KEY", ""),
		DeepSeekAPIKey: getEnv("DEEPSEEK_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
