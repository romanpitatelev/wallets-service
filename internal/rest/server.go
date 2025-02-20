package rest

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
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

func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		if err := s.server.Shutdown(context.Background()); err != nil {
			log.Warn().Err(err).Msg("failed to shutdown server")
		}
	}()

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start a server: %w", err)
	}

	return nil
}
