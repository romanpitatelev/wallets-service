package configs

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	DB DBConfig
}

type DBConfig struct {
	DSN string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Error().Msg("Error loading .env file, using default config")
	}

	return &Config{
		DB: DBConfig{
			DSN: os.Getenv("DSN"),
		},
	}
}
