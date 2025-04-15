package walletsservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/rs/zerolog/log"
)

const defaultRate = 1.0

type walletsStore interface {
	CreateWallet(ctx context.Context, wallet entity.Wallet, userID entity.UserID) (entity.Wallet, error)
	GetWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) (entity.Wallet, error)
	UpdateWallet(ctx context.Context, walletID entity.WalletID, updatedWallet entity.WalletUpdate, rate float64, userID entity.UserID) (entity.Wallet, error)
	DeleteWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) error
	GetWallets(ctx context.Context, request entity.GetWalletsRequest, userID entity.UserID) ([]entity.Wallet, error)
	ArchiveStaleWallets(ctx context.Context, checkPeriod time.Duration) error
}

type tx interface {
	DoWithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type xrClient interface {
	GetRate(ctx context.Context, from string, to string) (float64, error)
}

type Config struct {
	StaleWalletDuration time.Duration
	PerformCheckPeriod  time.Duration
}

type Service struct {
	cfg          Config
	walletsStore walletsStore
	xrClient     xrClient
	tx           tx
	metrics      *metrics
}

func New(cfg Config, walletsStore walletsStore, xrClient xrClient, tx tx) *Service {
	return &Service{
		cfg:          cfg,
		walletsStore: walletsStore,
		xrClient:     xrClient,
		tx:           tx,
		metrics:      newMetrics(),
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
			if err := s.walletsStore.ArchiveStaleWallets(ctx, s.cfg.PerformCheckPeriod); err != nil {
				return fmt.Errorf("error while archiving inactive wallets: %w", err)
			}
		}
	}
}

func (s *Service) CreateWallet(ctx context.Context, wallet entity.Wallet, userID entity.UserID) (entity.Wallet, error) {
	wallet, err := s.walletsStore.CreateWallet(ctx, wallet, userID)
	if err != nil {
		return entity.Wallet{}, fmt.Errorf("failed to create wallet: %w", err)
	}

	dbWallet, err := s.walletsStore.GetWallet(ctx, wallet.WalletID, userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to verify wallet creation")
	} else {
		log.Debug().Interface("dbWallet", dbWallet).Msg("Wallet created successfully")
	}

	return wallet, nil
}

func (s *Service) GetWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) (entity.Wallet, error) {
	wallet, err := s.walletsStore.GetWallet(ctx, walletID, userID)
	if err != nil {
		return entity.Wallet{}, fmt.Errorf("failed to get wallet: %w", err)
	}

	return wallet, nil
}

func (s *Service) UpdateWallet(ctx context.Context, walletID entity.WalletID, newInfoWallet entity.WalletUpdate, userID entity.UserID) (entity.Wallet, error) {
	timeStart := time.Now()

	var err error
	defer func() {
		if err != nil {
			s.metrics.updateFailed.WithLabelValues("update").Inc()
		} else {
			s.metrics.updateCompleted.WithLabelValues("update").Inc()
			s.metrics.updateDuration.WithLabelValues("update").Observe(time.Since(timeStart).Seconds())
		}
	}()

	var updatedWallet entity.Wallet

	if err := s.tx.DoWithTx(ctx, func(ctx context.Context) error {
		dbWallet, err := s.walletsStore.GetWallet(ctx, walletID, userID)
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

		updatedWallet, err = s.walletsStore.UpdateWallet(ctx, walletID, newInfoWallet, rate, userID)
		if err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}

		return nil
	}); err != nil {
		return entity.Wallet{}, fmt.Errorf("error in DoWithTX(): %w", err)
	}

	return updatedWallet, nil
}

func (s *Service) DeleteWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) error {
	if err := s.walletsStore.DeleteWallet(ctx, walletID, userID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

func (s *Service) GetAllWallets(ctx context.Context, request entity.GetWalletsRequest, userID entity.UserID) ([]entity.Wallet, error) {
	wallets, err := s.walletsStore.GetWallets(ctx, request, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting wallets info: %w", err)
	}

	return wallets, nil
}
