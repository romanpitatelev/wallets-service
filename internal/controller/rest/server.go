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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const ReadHeaderTimeoutValue = 3

type Config struct {
	Port int
}

type Server struct {
	server              *http.Server
	walletsHandler      walletsHandler
	transactionsHandler transactionsHandler
	port                int
	key                 *rsa.PublicKey
	metrics             *metrics
}

type walletsHandler interface {
	CreateWallet(w http.ResponseWriter, r *http.Request)
	GetWallet(w http.ResponseWriter, r *http.Request)
	UpdateWallet(w http.ResponseWriter, r *http.Request)
	DeleteWallet(w http.ResponseWriter, r *http.Request)
	GetWallets(w http.ResponseWriter, r *http.Request)
}

type transactionsHandler interface {
	Deposit(w http.ResponseWriter, r *http.Request)
	Withdraw(w http.ResponseWriter, r *http.Request)
	Transfer(w http.ResponseWriter, r *http.Request)
	GetTransactions(w http.ResponseWriter, r *http.Request)
}

//nolint:whitespace
func New(
	cfg Config,
	walletsHandler walletsHandler,
	transactionsHandler transactionsHandler,
	key *rsa.PublicKey,
) *Server {
	router := chi.NewRouter()
	s := &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           router,
			ReadHeaderTimeout: ReadHeaderTimeoutValue * time.Second,
		},
		walletsHandler:      walletsHandler,
		transactionsHandler: transactionsHandler,
		port:                cfg.Port,
		key:                 key,
		metrics:             newMetrics(),
	}

	router.Get("/metrics", promhttp.Handler().ServeHTTP)

	router.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Use(middleware.Recoverer)
			r.Use(s.jwtAuth)
			r.Use(s.metricTrack)

			r.Post("/wallets", s.walletsHandler.CreateWallet)
			r.Get("/wallets/{walletId}", s.walletsHandler.GetWallet)
			r.Patch("/wallets/{walletId}", s.walletsHandler.UpdateWallet)
			r.Delete("/wallets/{walletId}", s.walletsHandler.DeleteWallet)
			r.Get("/wallets", s.walletsHandler.GetWallets)
			r.Put("/wallets/{walletId}/deposit", s.transactionsHandler.Deposit)
			r.Put("/wallets/{walletId}/withdrawal", s.transactionsHandler.Withdraw)
			r.Put("/wallets/{walletId}/transfer", s.transactionsHandler.Transfer)
			r.Get("/wallets/{walletId}/transactions", s.transactionsHandler.GetTransactions)
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
