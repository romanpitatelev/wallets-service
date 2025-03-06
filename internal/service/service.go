package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

type walletStore interface {
	CreateWallet(ctx context.Context, wallet models.Wallet) (models.Wallet, error)
	GetWallet(ctx context.Context, walletID uuid.UUID) (models.Wallet, error)
	UpdateWallet(ctx context.Context, walletID uuid.UUID, updatedWallet models.WalletUpdate) (models.Wallet, error)
	DeleteWallet(ctx context.Context, walletID uuid.UUID) error
	GetWallets(ctx context.Context) ([]models.Wallet, error)
	ArchiveStaleWallets(ctx context.Context) error
}

type Config struct {
	StaleWalletDuration time.Duration
	PerformCheckPeriod  time.Duration
}

type Service struct {
	cfg         Config
	walletStore walletStore
}

func New(walletStore walletStore, cfg Config) *Service {
	return &Service{
		cfg:         cfg,
		walletStore: walletStore,
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet models.Wallet) (models.Wallet, error) {
	log.Info().Str("walletID", wallet.WalletID.String()).Msg("Creating wallet")

	wallet, err := s.walletStore.CreateWallet(ctx, wallet)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to create wallet: %w", err)
	}

	dbWallet, err := s.walletStore.GetWallet(ctx, wallet.WalletID)
	if err != nil {
		log.Error().Err(err).Msg("failed to verify wallet creation")
	} else {
		log.Debug().Interface("dbWallet", dbWallet).Msg("Wallet created successfully")
	}

	return wallet, nil
}

func (s *Service) GetWallet(ctx context.Context, walletID uuid.UUID) (models.Wallet, error) {
	wallet, err := s.walletStore.GetWallet(ctx, walletID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to get wallet: %w", err)
	}

	return wallet, nil
}

func (s *Service) UpdateWallet(ctx context.Context, walletID uuid.UUID, newInfoWallet models.WalletUpdate) (models.Wallet, error) {
	log.Info().Str("walletID", walletID.String()).Msg("Updating wallet")

	updatedWallet, err := s.walletStore.UpdateWallet(ctx, walletID, newInfoWallet)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to update wallet: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) DeleteWallet(ctx context.Context, walletID uuid.UUID) error {
	log.Info().Str("walletID", walletID.String()).Msg("Deleting wallet")

	if err := s.walletStore.DeleteWallet(ctx, walletID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

func (s *Service) GetAllWallets(ctx context.Context) ([]models.Wallet, error) {
	wallets, err := s.walletStore.GetWallets(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting wallets info: %w", err)
	}

	return wallets, nil
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.cfg.PerformCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.walletStore.ArchiveStaleWallets(ctx)
		}
	}
}
