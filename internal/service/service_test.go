//nolint:testpackage,dupl
package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/romanpitatelev/wallets-service/internal/service/mocks"
	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestDeposit(t *testing.T) {
	ctx := context.Background()
	userID := models.UserID(uuid.New())
	walletID := models.WalletID(uuid.New())
	now := time.Now()

	tests := []struct {
		name        string
		transaction models.Transaction
		mockWallet  models.Wallet
		mockRate    float64
		mockRateErr error
		setupMocks  func(*mocks.MockwalletStore, *mocks.MockxrClient, *mocks.MocktxProducer)
		expectedErr error
	}{
		{
			name: "successful deposit with same currency",
			transaction: models.Transaction{
				ToWalletID:  &walletID,
				Amount:      100.0,
				Currency:    "USD",
				CommittedAt: now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().Deposit(ctx, gomock.Any(), userID, 1.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "successful deposit with different currency",
			transaction: models.Transaction{
				ToWalletID:  &walletID,
				Amount:      100.0,
				Currency:    "EUR",
				CommittedAt: now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRate: 1.11,
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "EUR", "USD").Return(1.11, nil)
				ws.EXPECT().Deposit(ctx, gomock.Any(), userID, 1.11).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "exchange rate error",
			transaction: models.Transaction{
				ToWalletID:  &walletID,
				Amount:      100.0,
				Currency:    "NIO",
				CommittedAt: now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRateErr: models.ErrWrongCurrency,
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "NIO", "USD").Return(0.0, models.ErrWrongCurrency)
			},
			expectedErr: models.ErrWrongCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockWalletStore := mocks.NewMockwalletStore(ctrl)
			mockXRClient := mocks.NewMockxrClient(ctrl)
			mockTxProducer := mocks.NewMocktxProducer(ctrl)

			tt.setupMocks(mockWalletStore, mockXRClient, mockTxProducer)

			svc := &Service{
				walletStore: mockWalletStore,
				xrClient:    mockXRClient,
				producer:    mockTxProducer,
			}

			err := svc.Deposit(ctx, tt.transaction, userID)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:funlen
func TestWithdraw(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := models.UserID(uuid.New())
	walletID := models.WalletID(uuid.New())
	now := time.Now()

	tests := []struct {
		name        string
		transaction models.Transaction
		mockWallet  models.Wallet
		mockRate    float64
		mockRateErr error
		setupMocks  func(*mocks.MockwalletStore, *mocks.MockxrClient, *mocks.MocktxProducer)
		expectedErr error
	}{
		{
			name: "successful withdrawal with same currency",
			transaction: models.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().Withdraw(ctx, gomock.Any(), userID, 1.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "successful withdrawal with different currency",
			transaction: models.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "EUR",
				CommittedAt:  now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRate: 1.11,
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "EUR", "USD").Return(1.11, nil)
				ws.EXPECT().Withdraw(ctx, gomock.Any(), userID, 1.11).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "insufficient funds with same currency",
			transaction: models.Transaction{
				FromWalletID: &walletID,
				Amount:       600.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
			},
			expectedErr: models.ErrInsufficientFunds,
		},
		{
			name: "insufficient funds with foreign currency",
			transaction: models.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "RUB",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "RUB",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "USD", "RUB").Return(90.0, nil)
			},
			expectedErr: models.ErrInsufficientFunds,
		},
		{
			name: "exchange rate error",
			transaction: models.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "RUS",
				CommittedAt:  now,
			},
			mockWallet: models.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRateErr: models.ErrWrongCurrency,
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(models.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "RUS", "USD").Return(0.0, models.ErrWrongCurrency)
			},
			expectedErr: models.ErrWrongCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockWalletStore := mocks.NewMockwalletStore(ctrl)
			mockXRClient := mocks.NewMockxrClient(ctrl)
			mockTxProducer := mocks.NewMocktxProducer(ctrl)

			tt.setupMocks(mockWalletStore, mockXRClient, mockTxProducer)

			svc := &Service{
				walletStore: mockWalletStore,
				xrClient:    mockXRClient,
				producer:    mockTxProducer,
			}

			err := svc.Withdraw(ctx, tt.transaction, userID)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:funlen,maintidx
func TestTransfer(t *testing.T) {
	ctx := context.Background()
	userID := models.UserID(uuid.New())
	fromWalletID := models.WalletID(uuid.New())
	toWalletID := models.WalletID(uuid.New())
	now := time.Now()

	tests := []struct {
		name           string
		transaction    models.Transaction
		fromMockWallet models.Wallet
		toMockWallet   models.Wallet
		mockRate       float64
		mockRateErr    error
		setupMocks     func(*mocks.MockwalletStore, *mocks.MockxrClient, *mocks.MocktxProducer)
		expectedErr    error
	}{
		{
			name: "successful transfer with same currency",
			transaction: models.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       100.0,
				Currency:     "CHF",
				CommittedAt:  now,
			},
			fromMockWallet: models.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "CHF",
				Balance:  500.0,
			},
			toMockWallet: models.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "CHF",
				Balance:  200.0,
			},
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(models.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "CHF",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(models.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "CHF",
					Balance:  200.0,
				}, nil)
				ws.EXPECT().Transfer(ctx, gomock.Any(), userID, 1.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "successful transfer with different currency",
			transaction: models.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       10.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			fromMockWallet: models.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  60.0,
			},
			toMockWallet: models.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "RUB",
				Balance:  200.0,
			},
			mockRate: 90.0,
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(models.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  60.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(models.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "RUB",
					Balance:  200.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "USD", "RUB").Return(90.0, nil)
				ws.EXPECT().Transfer(ctx, gomock.Any(), userID, 90.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "insufficient funds in source wallet: same currency",
			transaction: models.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       600.0,
				Currency:     "CNY",
				CommittedAt:  now,
			},
			fromMockWallet: models.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  500.0,
			},
			toMockWallet: models.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  200.0,
			},
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(models.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(models.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  200.0,
				}, nil)
			},
			expectedErr: models.ErrInsufficientFunds,
		},
		{
			name: "insufficient funds in source wallet: same currency",
			transaction: models.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       600.0,
				Currency:     "CNY",
				CommittedAt:  now,
			},
			fromMockWallet: models.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  500.0,
			},
			toMockWallet: models.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  200.0,
			},
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(models.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(models.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  200.0,
				}, nil)
			},
			expectedErr: models.ErrInsufficientFunds,
		},
		{
			name: "exchange rate error",
			transaction: models.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       100.0,
				Currency:     "RSD",
				CommittedAt:  now,
			},
			fromMockWallet: models.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "RSD",
				Balance:  500.0,
			},
			toMockWallet: models.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "JPY",
				Balance:  200.0,
			},
			mockRateErr: models.ErrWrongCurrency,
			setupMocks: func(ws *mocks.MockwalletStore, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				ws.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(models.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "RSD",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(models.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "JPY",
					Balance:  200.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "RSD", "JPY").Return(0.0, models.ErrWrongCurrency)
			},
			expectedErr: models.ErrWrongCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockWalletStore := mocks.NewMockwalletStore(ctrl)
			mockXRClient := mocks.NewMockxrClient(ctrl)
			mockTxProducer := mocks.NewMocktxProducer(ctrl)

			tt.setupMocks(mockWalletStore, mockXRClient, mockTxProducer)

			svc := &Service{
				walletStore: mockWalletStore,
				xrClient:    mockXRClient,
				producer:    mockTxProducer,
			}

			err := svc.Transfer(ctx, tt.transaction, userID)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
