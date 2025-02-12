package service

import (
	"context"
	"fmt"
	"time"
)

type addrStore interface {
	Add(ctx context.Context, ipAddress string) error
	GetVisitsAll(ctx context.Context) (map[string]int, error)
}

type Service struct {
	addrStore addrStore
}

func New(addrStore addrStore) *Service {
	return &Service{
		addrStore: addrStore,
	}
}

func (s *Service) Add(ctx context.Context, ipAddress string) (time.Time, error) {
	if err := s.addrStore.Add(ctx, ipAddress); err != nil {
		return time.Time{}, fmt.Errorf("failed to add ip address: %w", err)
	}

	return time.Now(), nil
}

func (s *Service) GetVisitsAll(ctx context.Context) (map[string]int, error) {
	m, err := s.addrStore.GetVisitsAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get visits all: %w", err)
	}

	return m, nil
}
