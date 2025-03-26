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

//nolint:interfacebloat
type walletStore interface {
	CreateWallet(ctx context.Context, wallet models.Wallet, userID uuid.UUID) (models.Wallet, error)
	GetWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) (models.Wallet, error)
	UpdateWallet(ctx context.Context, walletID uuid.UUID, updatedWallet models.WalletUpdate, rate float64, userID uuid.UUID) (models.Wallet, error)
	DeleteWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) error
	GetWallets(ctx context.Context, request models.GetWalletsRequest, userID uuid.UUID) ([]models.Wallet, error)
	ArchiveStaleWallets(ctx context.Context, checkPeriod time.Duration) error
	DoWithTx(ctx context.Context, fn func(ctx context.Context) error) error
	Deposit(ctx context.Context, transaction models.Transaction, userID uuid.UUID, rate float64) error
	WithdrawFunds(ctx context.Context, transaction models.Transaction, userID uuid.UUID, rate float64) error
	Transfer(ctx context.Context, transaction models.Transaction, userID uuid.UUID, rate float64) error
	GetTransactions(ctx context.Context, request models.GetWalletsRequest, walletID uuid.UUID) ([]models.Transaction, error)
}

type xrClient interface {
	GetRate(ctx context.Context, from string, to string) (float64, error)
}

type txProducer interface {
	SendTxToKafka(transaction models.Transaction) error
}

type Config struct {
	StaleWalletDuration time.Duration
	PerformCheckPeriod  time.Duration
}

type Service struct {
	cfg         Config
	walletStore walletStore
	xrClient    xrClient
	producer    txProducer
}

func New(walletStore walletStore, cfg Config, xrClient xrClient, producer txProducer) *Service {
	return &Service{
		cfg:         cfg,
		walletStore: walletStore,
		xrClient:    xrClient,
		producer:    producer,
	}
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

	var updatedWallet models.Wallet

	if err := s.walletStore.DoWithTx(ctx, func(ctx context.Context) error {
		dbWallet, err := s.walletStore.GetWallet(ctx, walletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		if newInfoWallet.WalletName == "" {
			newInfoWallet.WalletName = dbWallet.WalletName
		}

		if newInfoWallet.Currency == "" {
			newInfoWallet.Currency = dbWallet.Currency
		}

		rate := defaultRate

		if dbWallet.Currency != strings.ToUpper(newInfoWallet.Currency) {
			rate, err = s.xrClient.GetRate(ctx, dbWallet.Currency, newInfoWallet.Currency)
			if err != nil {
				return fmt.Errorf("failed to obtain exchange rate: %w", err)
			}
		}

		updatedWallet, err = s.walletStore.UpdateWallet(ctx, walletID, newInfoWallet, rate, userID)
		if err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}

		return nil
	}); err != nil {
		return models.Wallet{}, fmt.Errorf("error in DoWithTX(): %w", err)
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

func (s *Service) Deposit(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error {
	if err := s.walletStore.DoWithTx(ctx, func(ctx context.Context) error {
		dbWallet, err := s.walletStore.GetWallet(ctx, transaction.ToWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		rate := defaultRate

		if dbWallet.Currency != strings.ToUpper(transaction.Currency) {
			rate, err = s.xrClient.GetRate(ctx, transaction.Currency, dbWallet.Currency)

			log.Debug().Msgf("exchange rate for transaction: %v", rate)
			if err != nil {
				return fmt.Errorf("failed to obtain exchange rate: %w", err)
			}
		}

		if err := s.walletStore.Deposit(ctx, transaction, userID, rate); err != nil {
			return fmt.Errorf("failed deposit: %w", err)
		}

		if err := s.producer.SendTxToKafka(transaction); err != nil {
			return fmt.Errorf("failed to produce deposit transaction: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error in DoWithTX(): %w", err)
	}

	return nil
}

func (s *Service) WithdrawFunds(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error {
	if err := s.walletStore.DoWithTx(ctx, func(ctx context.Context) error {
		dbWallet, err := s.walletStore.GetWallet(ctx, transaction.FromWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		rate := defaultRate

		if dbWallet.Currency != strings.ToUpper(transaction.Currency) {
			rate, err = s.xrClient.GetRate(ctx, transaction.Currency, dbWallet.Currency)
			if err != nil {
				return fmt.Errorf("failed to obtain exchange rate: %w", err)
			}
		}

		if dbWallet.Balance < transaction.Amount*rate {
			return models.ErrInsufficientFunds
		}

		if err := s.walletStore.WithdrawFunds(ctx, transaction, userID, rate); err != nil {
			return fmt.Errorf("failed withdrawal: %w", err)
		}

		if err := s.producer.SendTxToKafka(transaction); err != nil {
			return fmt.Errorf("failed to produce withdrawFunds transaction: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error in DoWithTX(): %w", err)
	}

	return nil
}

func (s *Service) Transfer(ctx context.Context, transaction models.Transaction, userID uuid.UUID) error {
	if err := s.walletStore.DoWithTx(ctx, func(ctx context.Context) error {
		dbFromTransferWallet, err := s.walletStore.GetWallet(ctx, transaction.FromWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		dbToTransferWallet, err := s.walletStore.GetWallet(ctx, transaction.ToWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		if strings.ToUpper(transaction.Currency) != dbFromTransferWallet.Currency {
			return models.ErrWrongCurrency
		}

		rate := defaultRate

		if dbFromTransferWallet.Currency != dbToTransferWallet.Currency {
			rate, err = s.xrClient.GetRate(ctx, dbFromTransferWallet.Currency, dbToTransferWallet.Currency)
			if err != nil {
				return fmt.Errorf("failed to obtain exchange rate: %w", err)
			}
		}

		if dbFromTransferWallet.Currency == strings.ToUpper(transaction.Currency) {
			if dbFromTransferWallet.Balance < transaction.Amount {
				return models.ErrInsufficientFunds
			}
		}

		if err := s.walletStore.Transfer(ctx, transaction, userID, rate); err != nil {
			return fmt.Errorf("transfer of funds failed: %w", err)
		}

		if err := s.producer.SendTxToKafka(transaction); err != nil {
			return fmt.Errorf("failed to produce transfer transaction: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error in DoWithTX(): %w", err)
	}

	return nil
}

func (s *Service) GetTransactions(ctx context.Context, request models.GetWalletsRequest, walletID uuid.UUID, userID uuid.UUID) ([]models.Transaction, error) {
	_, err := s.GetWallet(ctx, walletID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract wallet: %w", err)
	}

	transactions, err := s.walletStore.GetTransactions(ctx, request, walletID)
	if err != nil {
		return nil, fmt.Errorf("error getting all the transactions info: %w", err)
	}

	return transactions, nil
}
