package main

import (
	"github.com/romanpitatelev/wallets-service/internal/rest"

	"github.com/rs/zerolog/log"
)

func main() {

	server, err := rest.New()
	if err != nil {
		log.Error().Msg("Failed to create new server")
	}

	err = server.Run()
	if err != nil {
		panic(err)
	}
}
