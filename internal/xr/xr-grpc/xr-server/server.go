package xrgrpcserver

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/romanpitatelev/wallets-service/internal/models"
	xrgrpc "github.com/romanpitatelev/wallets-service/internal/xr/xr-grpc/gen/go"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Config struct {
	ListenAddress string
}

type Server struct {
	grpcServer *grpc.Server
	cfg        Config
	xrgrpc.UnimplementedExchangeRateServiceServer
}

func New(cfg Config) *Server {
	return &Server{
		grpcServer: grpc.NewServer(),
		cfg:        cfg,
	}
}

func (s *Server) GetRate(ctx context.Context, req *xrgrpc.RateRequest) (*xrgrpc.RateResponse, error) {
	exchangeRatesToRub := map[string]float64{
		"RUB": 1.0,
		"USD": 90.0,  //nolint:mnd
		"EUR": 100.0, //nolint:mnd
		"CNY": 12.3,  //nolint:mnd
		"CHF": 101.0, //nolint:mnd
		"GBP": 115.0, //nolint:mnd
		"KZT": 0.18,  //nolint:mnd
		"RSD": 0.83,  //nolint:mnd
	}

	fromXR, fromExists := exchangeRatesToRub[strings.ToUpper(req.GetFromCurrency())]
	toXR, toExists := exchangeRatesToRub[strings.ToUpper(req.GetToCurrency())]

	if !fromExists || !toExists {
		return nil, models.ErrWrongCurrency
	}

	rate := fromXR / toXR

	return &xrgrpc.RateResponse{Rate: rate}, nil
}

func (s *Server) Run(ctx context.Context) error {
	log.Info().Msgf("Starting grpc server on %s", s.cfg.ListenAddress)

	xrgrpc.RegisterExchangeRateServiceServer(s.grpcServer, s)
	reflection.Register(s.grpcServer)

	listener, err := net.Listen("tcp", s.cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("net.Listen(tcp, s.cfg.ListenAddress): %w", err)
	}

	go func() {
		<-ctx.Done()
		s.grpcServer.GracefulStop()
	}()

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("s.grpcServer(listener): %w", err)
	}

	return nil
}
