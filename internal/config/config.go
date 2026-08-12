package config

import (
	"errors"
	"log/slog"
	"os"
)

type Config struct {
	GeminiAPIKey        string
	DatabaseURL         string
	ListenAddr          string
	SGLangSocketPath    string
	LMCacheRedisURL     string
	EnableIngestion     bool
	Ephemeral           bool
	RouteLLMWeightsPath string
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func envBool(key string) bool {
	return os.Getenv(key) == "true" || os.Getenv(key) == "1"
}

func Load() (*Config, error) {
	c := &Config{
		GeminiAPIKey:        os.Getenv("GEMINI_API_KEY"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		ListenAddr:          envOrDefault("AIOF_LISTEN_ADDR", "0.0.0.0:8080"),
		SGLangSocketPath:    envOrDefault("SGLANG_SOCKET_PATH", "/tmp/sglang.sock"),
		LMCacheRedisURL:     os.Getenv("LMCACHE_REDIS_URL"),
		EnableIngestion:     envBool("AIOF_ENABLE_INGESTION"),
		Ephemeral:           envBool("AIOF_EPHEMERAL"),
		RouteLLMWeightsPath: envOrDefault("ROUTE_LLM_WEIGHTS_PATH", "config/route_llm_weights.json"),
	}

	if c.GeminiAPIKey == "" {
		return nil, errors.New("GEMINI_API_KEY is required")
	}

	if c.Ephemeral {
		slog.Warn("EPHEMERAL MODE: mutations are NOT persisted. Do not use in production.")
	} else if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required unless Ephemeral is true")
	}

	return c, nil
}
