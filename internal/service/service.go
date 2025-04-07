package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

const defaultRate = 1.0

//nolint:interfacebloat
type walletStore interface {
	CreateWallet(ctx context.Context, wallet models.Wallet, userID models.UserID) (models.Wallet, error)
	GetWallet(ctx context.Context, walletID models.WalletID, userID models.UserID) (models.Wallet, error)
	UpdateWallet(ctx context.Context, walletID models.WalletID, updatedWallet models.WalletUpdate, rate float64, userID models.UserID) (models.Wallet, error)
	DeleteWallet(ctx context.Context, walletID models.WalletID, userID models.UserID) error
	GetWallets(ctx context.Context, request models.GetWalletsRequest, userID models.UserID) ([]models.Wallet, error)
	ArchiveStaleWallets(ctx context.Context, checkPeriod time.Duration) error
	DoWithTx(ctx context.Context, fn func(ctx context.Context) error) error
	Deposit(ctx context.Context, transaction models.Transaction, userID models.UserID, rate float64) error
	Withdraw(ctx context.Context, transaction models.Transaction, userID models.UserID, rate float64) error
	Transfer(ctx context.Context, transaction models.Transaction, userID models.UserID, rate float64) error
	GetTransactions(ctx context.Context, request models.GetWalletsRequest, walletID models.WalletID, userID models.UserID) ([]models.Transaction, error)
}

type xrClient interface {
	GetRate(ctx context.Context, from string, to string) (float64, error)
}

//go:generate mockgen -source=service.go -destination=./mocks/transactions_mock.gen.go -package=mocks txProducer
type txProducer interface {
	ProduceTxToKafka(transaction models.Transaction) error
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
	metrics     *metrics
}

func New(cfg Config, walletStore walletStore, xrClient xrClient, producer txProducer) *Service {
	return &Service{
		cfg:         cfg,
		walletStore: walletStore,
		xrClient:    xrClient,
		producer:    producer,
		metrics:     newMetrics(),
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

func (s *Service) CreateWallet(ctx context.Context, wallet models.Wallet, userID models.UserID) (models.Wallet, error) {
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

func (s *Service) GetWallet(ctx context.Context, walletID models.WalletID, userID models.UserID) (models.Wallet, error) {
	wallet, err := s.walletStore.GetWallet(ctx, walletID, userID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to get wallet: %w", err)
	}

	return wallet, nil
}

func (s *Service) UpdateWallet(ctx context.Context, walletID models.WalletID, newInfoWallet models.WalletUpdate, userID models.UserID) (models.Wallet, error) {
	timeStart := time.Now()

	var err error
	defer func() {
		if err != nil {
			s.metrics.txFailed.WithLabelValues("update").Inc()
		} else {
			s.metrics.txCompleted.WithLabelValues("update").Inc()
			s.metrics.txDuration.WithLabelValues("update").Observe(time.Since(timeStart).Seconds())
		}
	}()

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

func (s *Service) DeleteWallet(ctx context.Context, walletID models.WalletID, userID models.UserID) error {
	if err := s.walletStore.DeleteWallet(ctx, walletID, userID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

func (s *Service) GetAllWallets(ctx context.Context, request models.GetWalletsRequest, userID models.UserID) ([]models.Wallet, error) {
	wallets, err := s.walletStore.GetWallets(ctx, request, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting wallets info: %w", err)
	}

	return wallets, nil
}

func (s *Service) Deposit(ctx context.Context, transaction models.Transaction, userID models.UserID) error {
	timeStart := time.Now()

	var err error
	defer func() {
		if err != nil {
			s.metrics.txFailed.WithLabelValues("deposit").Inc()
		} else {
			s.metrics.txCompleted.WithLabelValues("deposit").Inc()
			s.metrics.txDuration.WithLabelValues("deposit").Observe(time.Since(timeStart).Seconds())
		}
	}()

	if err := s.walletStore.DoWithTx(ctx, func(ctx context.Context) error {
		dbWallet, err := s.walletStore.GetWallet(ctx, *transaction.ToWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		rate := defaultRate

		if !strings.EqualFold(dbWallet.Currency, transaction.Currency) {
			rate, err = s.xrClient.GetRate(ctx, transaction.Currency, dbWallet.Currency)
			if err != nil {
				return fmt.Errorf("failed to obtain exchange rate: %w", err)
			}
		}

		if err := s.walletStore.Deposit(ctx, transaction, userID, rate); err != nil {
			return fmt.Errorf("failed deposit: %w", err)
		}

		if err := s.producer.ProduceTxToKafka(transaction); err != nil {
			return fmt.Errorf("failed to produce deposit transaction: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error in DoWithTX(): %w", err)
	}

	return nil
}

func (s *Service) Withdraw(ctx context.Context, transaction models.Transaction, userID models.UserID) error {
	timeStart := time.Now()

	var err error
	defer func() {
		if err != nil {
			s.metrics.txFailed.WithLabelValues("withdraw").Inc()
		} else {
			s.metrics.txCompleted.WithLabelValues("withdraw").Inc()
			s.metrics.txDuration.WithLabelValues("withdraw").Observe(time.Since(timeStart).Seconds())
		}
	}()

	if err := s.walletStore.DoWithTx(ctx, func(ctx context.Context) error {
		dbWallet, err := s.walletStore.GetWallet(ctx, *transaction.FromWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		rate := defaultRate

		if !strings.EqualFold(dbWallet.Currency, transaction.Currency) {
			rate, err = s.xrClient.GetRate(ctx, transaction.Currency, dbWallet.Currency)
			if err != nil {
				return fmt.Errorf("failed to obtain exchange rate: %w", err)
			}
		}

		if dbWallet.Balance < transaction.Amount*rate {
			return models.ErrInsufficientFunds
		}

		if err := s.walletStore.Withdraw(ctx, transaction, userID, rate); err != nil {
			return fmt.Errorf("failed withdrawal: %w", err)
		}

		if err := s.producer.ProduceTxToKafka(transaction); err != nil {
			return fmt.Errorf("failed to produce withdrawFunds transaction: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error in DoWithTX(): %w", err)
	}

	return nil
}

func (s *Service) Transfer(ctx context.Context, transaction models.Transaction, userID models.UserID) error {
	timeStart := time.Now()

	var err error
	defer func() {
		if err != nil {
			s.metrics.txFailed.WithLabelValues("transafer").Inc()
		} else {
			s.metrics.txCompleted.WithLabelValues("transfer").Inc()
			s.metrics.txDuration.WithLabelValues("transfer").Observe(time.Since(timeStart).Seconds())
		}
	}()

	if err := s.walletStore.DoWithTx(ctx, func(ctx context.Context) error {
		dbFromTransferWallet, err := s.walletStore.GetWallet(ctx, *transaction.FromWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		dbToTransferWallet, err := s.walletStore.GetWallet(ctx, *transaction.ToWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		if !strings.EqualFold(transaction.Currency, dbFromTransferWallet.Currency) {
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

		if err := s.producer.ProduceTxToKafka(transaction); err != nil {
			return fmt.Errorf("failed to produce transfer transaction: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error in DoWithTX(): %w", err)
	}

	return nil
}

//nolint:lll
func (s *Service) GetTransactions(ctx context.Context, request models.GetWalletsRequest, walletID models.WalletID, userID models.UserID) ([]models.Transaction, error) {
	transactions, err := s.walletStore.GetTransactions(ctx, request, walletID, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting all the transactions info: %w", err)
	}

	return transactions, nil
}
