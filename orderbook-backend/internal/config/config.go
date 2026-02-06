package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the orderbook backend
type Config struct {
	// Server settings
	ServerPort string

	// Yellow Network settings
	YellowNodeURL   string
	PrivateKey      string
	AdjudicatorAddr string

	// Trading settings
	DefaultToken string
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		YellowNodeURL:   getEnv("YELLOW_NODE_URL", "wss://clearnet.yellow.com/ws"),
		PrivateKey:      getEnv("PRIVATE_KEY", ""),
		AdjudicatorAddr: getEnv("ADJUDICATOR_ADDR", "0x33eA68432d7657CA49Db36f378A95c6c71d3BDF1"),
		DefaultToken:    getEnv("DEFAULT_TOKEN", "0x0000000000000000000000000000000000000000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
