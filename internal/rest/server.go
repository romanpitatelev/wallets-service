package rest

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/romanpitatelev/wallets-service/internal/currtime"
)

const ReadHeaderTimeoutValue = 3

type Server struct {
	router *chi.Mux
	server *http.Server
}

func New(pool *pgxpool.Pool) (*Server, error) {
	router := chi.NewRouter()
	s := &Server{
		router: router,
		server: &http.Server{
			Addr:              ":8081",
			Handler:           router,
			ReadHeaderTimeout: ReadHeaderTimeoutValue * time.Second,
		},
	}

	currtime.NewTimeHandler(router, pool)

	return s, nil
}

func (s *Server) Run() error {
	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start a server: %w", err)
	}

	return nil
}
