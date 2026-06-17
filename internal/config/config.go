package config

import "os"

type Config struct {
	Port                       string
	DatabaseURL                string
	DeepSeekAPIKey             string
	JWTSecret                  string
	YandexVisionAPIKey         string
    YandexFolderID             string
}

func Load() *Config {
	return &Config{
		Port:                     getEnv("PORT", "8080"),
		DatabaseURL:              getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/redpen?sslmode=disable"),
		DeepSeekAPIKey:           getEnv("DEEPSEEK_API_KEY", ""),
		JWTSecret:                getEnv("JWT_SECRET", "redpen-secret-change-me"),
		YandexVisionAPIKey:       getEnv("YANDEX_VISION_API_KEY", ""),
		YandexFolderID:           getEnv("YANDEX_FOLDER_ID", ""),
	}
}


func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
