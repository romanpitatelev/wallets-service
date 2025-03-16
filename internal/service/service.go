package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

const defaultRate = 1.0

type walletStore interface {
	CreateWallet(ctx context.Context, wallet models.Wallet, userID uuid.UUID) (models.Wallet, error)
	GetWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) (models.Wallet, error)
	UpdateWallet(ctx context.Context, walletID uuid.UUID, updatedWallet models.WalletUpdate, rate float64, userID uuid.UUID) (models.Wallet, error)
	DeleteWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) error
	GetWallets(ctx context.Context, request models.GetWalletsRequest, userID uuid.UUID) ([]models.Wallet, error)
	ArchiveStaleWallets(ctx context.Context, checkPeriod time.Duration) error
}

type xrClient interface {
	GetRate(ctx context.Context, from string, to string) (float64, error)
}

type Config struct {
	StaleWalletDuration time.Duration
	PerformCheckPeriod  time.Duration
}

type Service struct {
	cfg         Config
	walletStore walletStore
	xrClient    xrClient
}

func New(walletStore walletStore, cfg Config, xrClient xrClient) *Service {
	return &Service{
		cfg:         cfg,
		walletStore: walletStore,
		xrClient:    xrClient,
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet models.Wallet, userID uuid.UUID) (models.Wallet, error) {
	log.Info().Str("walletID", wallet.WalletID.String()).Msg("Creating wallet")

	wallet, err := s.walletStore.CreateWallet(ctx, wallet, userID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to create wallet: %w", err)
	}

	dbWallet, err := s.walletStore.GetWallet(ctx, wallet.WalletID, userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to verify wallet creation")
	} else {
		log.Debug().Interface("dbWallet", dbWallet).Msg("Wallet created successfully")
	}

	return wallet, nil
}

func (s *Service) GetWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) (models.Wallet, error) {
	wallet, err := s.walletStore.GetWallet(ctx, walletID, userID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to get wallet: %w", err)
	}

	return wallet, nil
}

func (s *Service) UpdateWallet(ctx context.Context, walletID uuid.UUID, newInfoWallet models.WalletUpdate, userID uuid.UUID) (models.Wallet, error) {
	log.Info().Str("walletID", walletID.String()).Msg("Updating wallet")

	dbWallet, err := s.walletStore.GetWallet(ctx, walletID, userID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("wallet not found: %w", err)
	}

	rate := defaultRate

	if dbWallet.Currency != strings.ToUpper(newInfoWallet.Currency) {
		rate, err = s.xrClient.GetRate(ctx, dbWallet.Currency, newInfoWallet.Currency)
		if err != nil {
			return models.Wallet{}, fmt.Errorf("failed to obtain exchange rate: %w", err)
		}
	}

	updatedWallet, err := s.walletStore.UpdateWallet(ctx, walletID, newInfoWallet, rate, userID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to update wallet: %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) DeleteWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) error {
	log.Info().Str("walletID", walletID.String()).Msg("Deleting wallet")

	if err := s.walletStore.DeleteWallet(ctx, walletID, userID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

func (s *Service) GetAllWallets(ctx context.Context, request models.GetWalletsRequest, userID uuid.UUID) ([]models.Wallet, error) {
	wallets, err := s.walletStore.GetWallets(ctx, request, userID)
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
			if err := s.walletStore.ArchiveStaleWallets(ctx, s.cfg.PerformCheckPeriod); err != nil {
				return fmt.Errorf("error while archiving inactive wallets: %w", err)
			}
		}
	}
}
