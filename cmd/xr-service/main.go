package main

import (
	"context"
	"os/signal"
	"syscall"

	xrserver "github.com/romanpitatelev/wallets-service/internal/xr/xr-server"
	"github.com/rs/zerolog/log"
)

const port = 2607

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server := xrserver.New(port)

	if err := server.Run(ctx); err != nil {
		log.Error().Err(err).Msg("failed to run xr server")
	}
}
