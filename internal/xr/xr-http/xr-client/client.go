package xrhttpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

const route = "/api/v1/xr?from=%v&to=%v"

type Client struct {
	cfg Config
}

type Config struct {
	ServerAddress string
}

func New(cfg Config) *Client {
	return &Client{
		cfg: cfg,
	}
}

func (c *Client) GetRate(ctx context.Context, from string, to string) (float64, error) {
	url := c.cfg.ServerAddress + fmt.Sprintf(route, from, to)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0.0, fmt.Errorf("xr client: failed to create request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0.0, fmt.Errorf("xr client: failed to send request: %w", err)
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close response body")
		}
	}()

	switch {
	case response.StatusCode == http.StatusUnprocessableEntity:
		return 0.0, models.ErrWrongCurrency
	case response.StatusCode != http.StatusOK:
		return 0.0, fmt.Errorf("status code not OK: %w", err)
	default:
		var resp models.XRResponse

		if err = json.NewDecoder(response.Body).Decode(&resp); err != nil {
			return 0.0, fmt.Errorf("xr client: error decoding response body: %w", err)
		}

		return resp.Rate, nil
	}
}
