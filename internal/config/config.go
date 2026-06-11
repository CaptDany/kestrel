package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port   int
	DBPath string
}

func Load() *Config {
	return &Config{
		Port:   getEnvInt("KESTREL_PORT", 8000),
		DBPath: getEnv("KESTREL_DB_PATH", "data/kestrel.db"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
