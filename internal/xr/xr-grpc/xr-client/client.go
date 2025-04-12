package xrgrpcclient

import (
	"context"
	"fmt"

	xrgrpc "github.com/romanpitatelev/wallets-service/internal/xr/xr-grpc/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Host string
}

type Client struct {
	cfg    Config
	client xrgrpc.ExchangeRateServiceClient
}

func New(cfg Config) (*Client, error) {
	client, err := grpc.NewClient(cfg.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient(): %w", err)
	}

	return &Client{
		cfg:    cfg,
		client: xrgrpc.NewExchangeRateServiceClient(client),
	}, nil
}

func (c *Client) GetRate(ctx context.Context, from string, to string) (float64, error) {
	response, err := c.client.GetRate(ctx, &xrgrpc.RateRequest{
		FromCurrency: from,
		ToCurrency:   to,
	})
	if err != nil {
		return 0.0, fmt.Errorf("gRPC client error: %w", err)
	}

	return response.GetRate(), nil
}
