package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/romanpitatelev/wallets-service/internal/ip"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	db, err := gorm.Open(postgres.Open(os.Getenv("DSN")), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&ip.IP{})
	if err != nil {
		log.Error().Msg("Automigration failure")
	}
}
