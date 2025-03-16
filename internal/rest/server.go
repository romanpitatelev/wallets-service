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
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
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

type service interface {
	CreateWallet(ctx context.Context, wallet models.Wallet, userID uuid.UUID) (models.Wallet, error)
	GetWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) (models.Wallet, error)
	UpdateWallet(ctx context.Context, walletID uuid.UUID, updatedWallet models.WalletUpdate, userID uuid.UUID) (models.Wallet, error)
	DeleteWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) error
	GetAllWallets(ctx context.Context, request models.GetWalletsRequest, userID uuid.UUID) ([]models.Wallet, error)
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

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Recoverer)
		r.Use(s.jwtAuth)

		r.Post("/wallets", s.CreateWallet)
		r.Get("/wallets/{walletId}", s.GetWallet)
		r.Patch("/wallets/{walletId}", s.UpdateWallet)
		r.Delete("/wallets/{walletId}", s.DeleteWallet)
		r.Get("/wallets", s.GetWallets)
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
