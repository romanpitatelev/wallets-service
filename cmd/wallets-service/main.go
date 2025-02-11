package main

import (
	"github.com/romanpitatelev/wallets-service/configs"
	"github.com/romanpitatelev/wallets-service/internal/rest"
	"github.com/romanpitatelev/wallets-service/pkg/db"
	"github.com/rs/zerolog/log"
)

func main() {
	conf := configs.LoadConfig()
	dbInstance := db.NewDb(conf)

	server, err := rest.New(dbInstance.DB)
	if err != nil {
		log.Error().Msg("Failed to create new server")
	}

	err = server.Run()
	if err != nil {
		panic(err)
	}
}
