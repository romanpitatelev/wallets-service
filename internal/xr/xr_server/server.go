package xrserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

const (
	XRReadHeaderTimeoutValue = 3
	roundNumber              = 10000
)

type Server struct {
	server *http.Server
	port   int
}

func New(port int) *Server {
	router := chi.NewRouter()
	s := &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           router,
			ReadHeaderTimeout: XRReadHeaderTimeoutValue * time.Second,
		},
		port: port,
	}

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/xr", s.GetExchangeRate)
	})

	return s
}

func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		if err := s.server.Shutdown(ctx); err != nil {
			log.Warn().Err(err).Msg("failed to shutdown xr server")
		}
	}()

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start xr server: %w", err)
	}

	return nil
}

func (s *Server) GetExchangeRate(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	xr := models.XRRequest{
		FromCurrency: queryParams.Get("from"),
		ToCurrency:   queryParams.Get("to"),
	}

	exchangeRatesToRub := map[string]float64{
		"RUB": 1.0,
		"USD": 90.0,  //nolint:mnd
		"EUR": 100.0, //nolint:mnd
		"CNY": 12.0,  //nolint:mnd
		"CHF": 101.0, //nolint:mnd
		"GBP": 115.0, //nolint:mnd
		"KZT": 0.18,  //nolint:mnd
		"RSD": 0.83,  //nolint:mnd
	}

	fromXR, fromExists := exchangeRatesToRub[strings.ToUpper(xr.FromCurrency)]
	toXR, toExists := exchangeRatesToRub[strings.ToUpper(xr.ToCurrency)]

	if !fromExists || !toExists {
		s.errorResponse(w, "error getting exchange rate", models.ErrWrongCurrency)

		return
	}

	rate := fromXR / toXR
	rateRounded := math.Round(rate*roundNumber) / roundNumber

	response := models.XRResponse{Rate: rateRounded}

	s.okResponse(w, http.StatusOK, response)
}

func (s *Server) errorResponse(w http.ResponseWriter, errorText string, err error) {
	statusCode := http.StatusInternalServerError

	if errors.Is(err, models.ErrWrongCurrency) {
		statusCode = http.StatusNotFound
	}

	errResp := fmt.Errorf("%s: %w", errorText, err).Error()
	if statusCode == http.StatusInternalServerError {
		errResp = http.StatusText(http.StatusInternalServerError)

		log.Warn().Err(err).Msg("warning message")
	}

	response, err := json.Marshal(errResp)
	if err != nil {
		log.Warn().Err(err).Msg("error marshaling errorResponse")
	}

	w.WriteHeader(statusCode)

	if _, err := w.Write(response); err != nil {
		log.Warn().Err(err).Msg("error writing response")
	}
}

func (s *Server) okResponse(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Warn().Err(err).Msg("error encoding response")
	}
}
