package xrgrpcclient

import (
	"context"
	"fmt"

	xr_grpc "github.com/romanpitatelev/wallets-service/internal/xr/xr-grpc/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Host string
}

type Client struct {
	cfg    Config
	client xr_grpc.ExchangeRateServiceClient
}

func New(cfg Config) (*Client, error) {
	client, err := grpc.NewClient(cfg.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc.Dial: %w", err)
	}

	return &Client{
		cfg:    cfg,
		client: xr_grpc.NewExchangeRateServiceClient(client),
	}, nil
}

func (c *Client) GetRate(ctx context.Context, from string, to string) (float64, error) {
	response, err := c.client.GetRate(ctx, &xr_grpc.RateRequest{
		FromCurrency: from,
		ToCurrency:   to,
	})
	if err != nil {
		return 0.0, fmt.Errorf("GRPC client error: %w", err)
	}

	return response.GetRate(), nil
}
