package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const ReadHeaderTimeoutValue = 3

type Server struct {
	server  *http.Server
	service service
}

type service interface {
	Add(ctx context.Context, ipAddress string) (time.Time, error)
	GetVisitsAll(ctx context.Context) (map[string]int, error)
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

	router.Get("/time", s.TimeNow)
	router.Get("/visitors", s.GetVisitors)

	return s, nil
}

func (s *Server) Run() error {
	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start a server: %w", err)
	}

	return nil
}
