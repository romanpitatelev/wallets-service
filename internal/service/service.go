package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

type walletStore interface {
	CreateWallet(ctx context.Context, wallet models.Wallet) error
	GetWallet(ctx context.Context, walletID uuid.UUID) (*models.Wallet, error)
	UpdateWallet(ctx context.Context, wallet models.Wallet) error
	DeleteWallet(ctx context.Context, walletID uuid.UUID) error
	GetWallets(ctx context.Context, userID uuid.UUID) ([]models.Wallet, error)
}

type Service struct {
	walletStore walletStore
}

func New(walletStore walletStore) *Service {
	return &Service{
		walletStore: walletStore,
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet models.Wallet) error {
	log.Info().Str("walletID", wallet.WalletID.String()).Msg("Creating wallet")

	if err := s.walletStore.CreateWallet(ctx, wallet); err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	return nil
}

func (s *Service) GetWallet(ctx context.Context, walletID uuid.UUID) (*models.Wallet, error) {
	wallet, err := s.walletStore.GetWallet(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return wallet, nil
}

func (s *Service) UpdateWallet(ctx context.Context, wallet models.Wallet) error {
	log.Info().Str("walletID", wallet.WalletID.String()).Msg("Updating wallet")

	if err := s.walletStore.UpdateWallet(ctx, wallet); err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	return nil
}

func (s *Service) DeleteWallet(ctx context.Context, walletID uuid.UUID) error {
	log.Info().Str("walletID", walletID.String()).Msg("Deleting wallet")

	if err := s.walletStore.DeleteWallet(ctx, walletID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

func (s *Service) GetAllWallets(ctx context.Context, userID uuid.UUID) ([]models.Wallet, error) {
	wallets, err := s.walletStore.GetWallets(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting wallets info: %w", err)
	}

	return wallets, nil
}
