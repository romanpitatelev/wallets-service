package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

const ReadHeaderTimeoutValue = 3

type Server struct {
	server  *http.Server
	service service
}

type service interface {
	CreateWallet(ctx context.Context, wallet models.Wallet) error
	GetWallet(ctx context.Context, walletID uuid.UUID) (*models.Wallet, error)
	UpdateWallet(ctx context.Context, wallet models.Wallet) error
	DeleteWallet(ctx context.Context, walletID uuid.UUID) error
	GetAllWallets(ctx context.Context, userID uuid.UUID) ([]models.Wallet, error)
}

func New(service service) (*Server, error) {
	router := chi.NewRouter()
	s := &Server{
		service: service,
		server: &http.Server{
			Addr:              ":8081",
			Handler:           router,
			ReadHeaderTimeout: ReadHeaderTimeoutValue * time.Second,
		},
	}

	router.Route("/api/v1/wallets", func(r chi.Router) {
		r.Post("/", s.CreateWallet)
		r.Get("/{id}", s.GetWallet)
		r.Patch("/{id}", s.UpdateWallet)
		r.Delete("/{id}", s.DeleteWallet)
		r.Get("/", s.GetWallets)
	})

	return s, nil
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
