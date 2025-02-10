package rest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/romanpitatelev/wallets-service/internal/time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	router *chi.Mux
	server *http.Server
}

func New() (*Server, error) {

	router := chi.NewRouter()
	s := &Server{
		router: router,
		server: &http.Server{
			Addr:    ":8081",
			Handler: router,
		},
	}

	time.NewTimeHandler(router)

	return s, nil
}

func (s *Server) Run() error {
	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start a server: %w", err)
	}
	return nil
}
