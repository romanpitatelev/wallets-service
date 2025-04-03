package rest

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const ReadHeaderTimeoutValue = 3

type Config struct {
	Port int
}

type Server struct {
	server  *http.Server
	service service
	port    int
	key     *rsa.PublicKey
}

func New(conf Config, service service, key *rsa.PublicKey) *Server {
	router := chi.NewRouter()
	s := &Server{
		service: service,
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", conf.Port),
			Handler:           router,
			ReadHeaderTimeout: ReadHeaderTimeoutValue * time.Second,
		},
		port: conf.Port,
		key:  key,
	}

	router.Route("/api", func(r chi.Router) {
		
		router.Route("/v1", func(r chi.Router) {
			r.Use(middleware.Recoverer)
			r.Use(s.jwtAuth)

			r.Post("/wallets", s.createWallet)
			r.Get("/wallets/{walletId}", s.getWallet)
			r.Patch("/wallets/{walletId}", s.updateWallet)
			r.Delete("/wallets/{walletId}", s.deleteWallet)
			r.Get("/wallets", s.getWallets)
			r.Put("/wallets/{walletId}/deposit", s.deposit)
			r.Put("/wallets/{walletId}/withdrawal", s.withdraw)
			r.Put("/wallets/{walletId}/transfer", s.transfer)
			r.Get("/wallets/{walletId}/transactions", s.getTransactions)
		})
	})

	return s
}

func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		if err := s.server.Shutdown(ctx); err != nil {
			log.Warn().Err(err).Msg("failed to shutdown server")
		}
	}()

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start a server: %w", err)
	}

	return nil
}
