package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	xrgrpcserver "github.com/romanpitatelev/wallets-service/internal/xr/xr-grpc/xr-server"
	xrhttpserver "github.com/romanpitatelev/wallets-service/internal/xr/xr-http/xr-server"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	httpPort = 2607
	grpcPort = 2608
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	httpServer := xrhttpserver.New(httpPort)
	grpcServer := xrgrpcserver.New(xrgrpcserver.Config{
		ListenAddress: fmt.Sprintf(":%d", grpcPort),
	})

	errGr, ctx := errgroup.WithContext(ctx)

	errGr.Go(func() error {
		if err := httpServer.Run(ctx); err != nil {
			return fmt.Errorf("failed to run xr http server: %w", err)
		}

		return nil
	})

	errGr.Go(func() error {
		if err := grpcServer.Run(ctx); err != nil {
			return fmt.Errorf("failed to run xr gRPC server: %w", err)
		}

		return nil
	})

	if err := errGr.Wait(); err != nil {
		log.Panic().Err(err).Msg("failed to wait xr server blocks")
	}
}
