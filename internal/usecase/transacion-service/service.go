package transacionservice

import (
	"context"
	"fmt"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"strings"
)

const defaultRate = 1.0

type walletStore interface {
	GetWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) (entity.Wallet, error)
}

type tx interface {
	DoWithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type transactionsStore interface {
	Deposit(ctx context.Context, transaction entity.Transaction, userID entity.UserID, rate float64) error
	Withdraw(ctx context.Context, transaction entity.Transaction, userID entity.UserID, rate float64) error
	Transfer(ctx context.Context, transaction entity.Transaction, userID entity.UserID, rate float64) error
	GetTransactions(ctx context.Context, request entity.GetWalletsRequest, walletID entity.WalletID, userID entity.UserID) ([]entity.Transaction, error)
}

type xrClient interface {
	GetRate(ctx context.Context, from string, to string) (float64, error)
}

//go:generate mockgen -source=service.go -destination=./mocks/transactions_mock.gen.go -package=mocks txProducer
type txProducer interface {
	ProduceTxToKafka(transaction entity.Transaction) error
}

type Service struct {
	walletStore       walletStore
	transactionsStore transactionsStore
	xrClient          xrClient
	tx                tx
	producer          txProducer
	//metrics     *metrics
}

func New(
	walletStore walletStore,
	transactionsStore transactionsStore,
	tx tx,
	xrClient xrClient,
	producer txProducer,
) *Service {
	return &Service{
		walletStore:       walletStore,
		transactionsStore: transactionsStore,
		xrClient:          xrClient,
		tx:                tx,
		producer:          producer,
		//metrics:     newMetrics(),
	}
}

func (s *Service) Deposit(ctx context.Context, transaction entity.Transaction, userID entity.UserID) error {
	//timeStart := time.Now()
	//
	//var err error
	//defer func() {
	//	if err != nil {
	//		s.metrics.txFailed.WithLabelValues("deposit").Inc()
	//	} else {
	//		s.metrics.txCompleted.WithLabelValues("deposit").Inc()
	//		s.metrics.txDuration.WithLabelValues("deposit").Observe(time.Since(timeStart).Seconds())
	//	}
	//}()

	if err := s.tx.DoWithTx(ctx, func(ctx context.Context) error {
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

		if err := s.transactionsStore.Deposit(ctx, transaction, userID, rate); err != nil {
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

func (s *Service) Withdraw(ctx context.Context, transaction entity.Transaction, userID entity.UserID) error {
	//timeStart := time.Now()
	//
	//var err error
	//defer func() {
	//	if err != nil {
	//		s.metrics.txFailed.WithLabelValues("withdraw").Inc()
	//	} else {
	//		s.metrics.txCompleted.WithLabelValues("withdraw").Inc()
	//		s.metrics.txDuration.WithLabelValues("withdraw").Observe(time.Since(timeStart).Seconds())
	//	}
	//}()

	if err := s.tx.DoWithTx(ctx, func(ctx context.Context) error {
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
			return entity.ErrInsufficientFunds
		}

		if err := s.transactionsStore.Withdraw(ctx, transaction, userID, rate); err != nil {
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

func (s *Service) Transfer(ctx context.Context, transaction entity.Transaction, userID entity.UserID) error {
	// FIXME
	//timeStart := time.Now()
	//
	//var err error
	//defer func() {
	//	if err != nil {
	//		s.metrics.txFailed.WithLabelValues("transafer").Inc()
	//	} else {
	//		s.metrics.txCompleted.WithLabelValues("transfer").Inc()
	//		s.metrics.txDuration.WithLabelValues("transfer").Observe(time.Since(timeStart).Seconds())
	//	}
	//}()

	if err := s.tx.DoWithTx(ctx, func(ctx context.Context) error {
		dbFromTransferWallet, err := s.walletStore.GetWallet(ctx, *transaction.FromWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		dbToTransferWallet, err := s.walletStore.GetWallet(ctx, *transaction.ToWalletID, userID)
		if err != nil {
			return fmt.Errorf("wallet not found: %w", err)
		}

		if !strings.EqualFold(transaction.Currency, dbFromTransferWallet.Currency) {
			return entity.ErrWrongCurrency
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
				return entity.ErrInsufficientFunds
			}
		}

		if err := s.transactionsStore.Transfer(ctx, transaction, userID, rate); err != nil {
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

func (s *Service) GetTransactions(
	ctx context.Context,
	request entity.GetWalletsRequest,
	walletID entity.WalletID,
	userID entity.UserID,
) ([]entity.Transaction, error) {
	transactions, err := s.transactionsStore.GetTransactions(ctx, request, walletID, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting all the transactions info: %w", err)
	}

	return transactions, nil
}
