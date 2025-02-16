package configs

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	BindAddress      string
	PostgresHost     string
	PostgresPort     string
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string
}

func NewConfig() *Config {
	err := godotenv.Load("example.env")
	if err != nil {
		log.Panic().Msg("Error loading example.env file")
	}

	log.Debug().Msg("Environment variables loaded")

	config := &Config{
		BindAddress:      os.Getenv("BIND_ADDRESS"),
		PostgresHost:     os.Getenv("POSTGRES_HOST"),
		PostgresPort:     os.Getenv("POSTGRES_PORT"),
		PostgresDatabase: os.Getenv("POSTGRES_DATABASE"),
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
	}

	log.Debug().Msg("Loaded configuration")

	return config
}
