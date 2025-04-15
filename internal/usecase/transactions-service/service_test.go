//nolint:testpackage,dupl
package transactionsservice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/romanpitatelev/wallets-service/internal/usecase/transactions-service/mocks"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals
var (
	testMetrics *metrics
	metricsOnce sync.Once
)

func getTestMetrics() *metrics {
	metricsOnce.Do(func() {
		testMetrics = newMetrics()
	})

	return testMetrics
}

//nolint:funlen
func TestDeposit(t *testing.T) {
	ctx := context.Background()
	userID := entity.UserID(uuid.New())
	walletID := entity.WalletID(uuid.New())
	now := time.Now()

	tests := []struct {
		name        string
		transaction entity.Transaction
		mockWallet  entity.Wallet
		mockRate    float64
		mockRateErr error
		setupMocks  func(*mocks.MockwalletsStore, *mocks.MocktransactionsStore, *mocks.Mocktx, *mocks.MockxrClient, *mocks.MocktxProducer)
		expectedErr error
	}{
		{
			name: "successful deposit with same currency",
			transaction: entity.Transaction{
				ToWalletID:  &walletID,
				Amount:      100.0,
				Currency:    "USD",
				CommittedAt: now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				ts.EXPECT().Deposit(ctx, gomock.Any(), userID, 1.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "successful deposit with different currency",
			transaction: entity.Transaction{
				ToWalletID:  &walletID,
				Amount:      100.0,
				Currency:    "EUR",
				CommittedAt: now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRate: 1.11,
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "EUR", "USD").Return(1.11, nil)
				ts.EXPECT().Deposit(ctx, gomock.Any(), userID, 1.11).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "exchange rate error",
			transaction: entity.Transaction{
				ToWalletID:  &walletID,
				Amount:      100.0,
				Currency:    "NIO",
				CommittedAt: now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRateErr: entity.ErrWrongCurrency,
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "NIO", "USD").Return(0.0, entity.ErrWrongCurrency)
			},
			expectedErr: entity.ErrWrongCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockWalletsStore := mocks.NewMockwalletsStore(ctrl)
			mockTransactionsStore := mocks.NewMocktransactionsStore(ctrl)
			mockTx := mocks.NewMocktx(ctrl)
			mockXRClient := mocks.NewMockxrClient(ctrl)
			mockTxProducer := mocks.NewMocktxProducer(ctrl)

			tt.setupMocks(mockWalletsStore, mockTransactionsStore, mockTx, mockXRClient, mockTxProducer)

			svc := &Service{
				walletsStore:      mockWalletsStore,
				transactionsStore: mockTransactionsStore,
				tx:                mockTx,
				xrClient:          mockXRClient,
				producer:          mockTxProducer,
				metrics:           getTestMetrics(),
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
	userID := entity.UserID(uuid.New())
	walletID := entity.WalletID(uuid.New())
	now := time.Now()

	tests := []struct {
		name        string
		transaction entity.Transaction
		mockWallet  entity.Wallet
		mockRate    float64
		mockRateErr error
		setupMocks  func(*mocks.MockwalletsStore, *mocks.MocktransactionsStore, *mocks.Mocktx, *mocks.MockxrClient, *mocks.MocktxProducer)
		expectedErr error
	}{
		{
			name: "successful withdrawal with same currency",
			transaction: entity.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				ts.EXPECT().Withdraw(ctx, gomock.Any(), userID, 1.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "successful withdrawal with different currency",
			transaction: entity.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "EUR",
				CommittedAt:  now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRate: 1.11,
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "EUR", "USD").Return(1.11, nil)
				ts.EXPECT().Withdraw(ctx, gomock.Any(), userID, 1.11).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "insufficient funds with same currency",
			transaction: entity.Transaction{
				FromWalletID: &walletID,
				Amount:       600.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
			},
			expectedErr: entity.ErrInsufficientFunds,
		},
		{
			name: "insufficient funds with foreign currency",
			transaction: entity.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "RUB",
				Balance:  500.0,
			},
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "RUB",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "USD", "RUB").Return(90.0, nil)
			},
			expectedErr: entity.ErrInsufficientFunds,
		},
		{
			name: "exchange rate error",
			transaction: entity.Transaction{
				FromWalletID: &walletID,
				Amount:       100.0,
				Currency:     "RUS",
				CommittedAt:  now,
			},
			mockWallet: entity.Wallet{
				WalletID: walletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  500.0,
			},
			mockRateErr: entity.ErrWrongCurrency,
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, walletID, userID).Return(entity.Wallet{
					WalletID: walletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  500.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "RUS", "USD").Return(0.0, entity.ErrWrongCurrency)
			},
			expectedErr: entity.ErrWrongCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockWalletsStore := mocks.NewMockwalletsStore(ctrl)
			mockTransactionsStore := mocks.NewMocktransactionsStore(ctrl)
			mockTx := mocks.NewMocktx(ctrl)
			mockXRClient := mocks.NewMockxrClient(ctrl)
			mockTxProducer := mocks.NewMocktxProducer(ctrl)

			tt.setupMocks(mockWalletsStore, mockTransactionsStore, mockTx, mockXRClient, mockTxProducer)

			svc := &Service{
				walletsStore:      mockWalletsStore,
				transactionsStore: mockTransactionsStore,
				xrClient:          mockXRClient,
				producer:          mockTxProducer,
				metrics:           getTestMetrics(),
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
	userID := entity.UserID(uuid.New())
	fromWalletID := entity.WalletID(uuid.New())
	toWalletID := entity.WalletID(uuid.New())
	now := time.Now()

	tests := []struct {
		name           string
		transaction    entity.Transaction
		fromMockWallet entity.Wallet
		toMockWallet   entity.Wallet
		mockRate       float64
		mockRateErr    error
		setupMocks     func(*mocks.MockwalletsStore, *mocks.MocktransactionsStore, *mocks.Mocktx, *mocks.MockxrClient, *mocks.MocktxProducer)
		expectedErr    error
	}{
		{
			name: "successful transfer with same currency",
			transaction: entity.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       100.0,
				Currency:     "CHF",
				CommittedAt:  now,
			},
			fromMockWallet: entity.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "CHF",
				Balance:  500.0,
			},
			toMockWallet: entity.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "CHF",
				Balance:  200.0,
			},
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(entity.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "CHF",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(entity.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "CHF",
					Balance:  200.0,
				}, nil)
				ts.EXPECT().Transfer(ctx, gomock.Any(), userID, 1.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "successful transfer with different currency",
			transaction: entity.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       10.0,
				Currency:     "USD",
				CommittedAt:  now,
			},
			fromMockWallet: entity.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "USD",
				Balance:  60.0,
			},
			toMockWallet: entity.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "RUB",
				Balance:  200.0,
			},
			mockRate: 90.0,
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(entity.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "USD",
					Balance:  60.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(entity.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "RUB",
					Balance:  200.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "USD", "RUB").Return(90.0, nil)
				ts.EXPECT().Transfer(ctx, gomock.Any(), userID, 90.0).Return(nil)
				tp.EXPECT().ProduceTxToKafka(gomock.Any()).Return(nil)
			},
		},
		{
			name: "insufficient funds in source wallet: same currency",
			transaction: entity.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       600.0,
				Currency:     "CNY",
				CommittedAt:  now,
			},
			fromMockWallet: entity.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  500.0,
			},
			toMockWallet: entity.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  200.0,
			},
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(entity.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(entity.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  200.0,
				}, nil)
			},
			expectedErr: entity.ErrInsufficientFunds,
		},
		{
			name: "insufficient funds in source wallet: same currency",
			transaction: entity.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       600.0,
				Currency:     "CNY",
				CommittedAt:  now,
			},
			fromMockWallet: entity.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  500.0,
			},
			toMockWallet: entity.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "CNY",
				Balance:  200.0,
			},
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(entity.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(entity.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "CNY",
					Balance:  200.0,
				}, nil)
			},
			expectedErr: entity.ErrInsufficientFunds,
		},
		{
			name: "exchange rate error",
			transaction: entity.Transaction{
				FromWalletID: &fromWalletID,
				ToWalletID:   &toWalletID,
				Amount:       100.0,
				Currency:     "RSD",
				CommittedAt:  now,
			},
			fromMockWallet: entity.Wallet{
				WalletID: fromWalletID,
				UserID:   userID,
				Currency: "RSD",
				Balance:  500.0,
			},
			toMockWallet: entity.Wallet{
				WalletID: toWalletID,
				UserID:   userID,
				Currency: "JPY",
				Balance:  200.0,
			},
			mockRateErr: entity.ErrWrongCurrency,
			setupMocks: func(ws *mocks.MockwalletsStore, ts *mocks.MocktransactionsStore, tx *mocks.Mocktx, xr *mocks.MockxrClient, tp *mocks.MocktxProducer) {
				tx.EXPECT().DoWithTx(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						return fn(ctx)
					},
				)
				ws.EXPECT().GetWallet(ctx, fromWalletID, userID).Return(entity.Wallet{
					WalletID: fromWalletID,
					UserID:   userID,
					Currency: "RSD",
					Balance:  500.0,
				}, nil)
				ws.EXPECT().GetWallet(ctx, toWalletID, userID).Return(entity.Wallet{
					WalletID: toWalletID,
					UserID:   userID,
					Currency: "JPY",
					Balance:  200.0,
				}, nil)
				xr.EXPECT().GetRate(ctx, "RSD", "JPY").Return(0.0, entity.ErrWrongCurrency)
			},
			expectedErr: entity.ErrWrongCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockWalletsStore := mocks.NewMockwalletsStore(ctrl)
			mockTransactionsStore := mocks.NewMocktransactionsStore(ctrl)
			mockTx := mocks.NewMocktx(ctrl)
			mockXRClient := mocks.NewMockxrClient(ctrl)
			mockTxProducer := mocks.NewMocktxProducer(ctrl)

			tt.setupMocks(mockWalletsStore, mockTransactionsStore, mockTx, mockXRClient, mockTxProducer)

			svc := &Service{
				walletsStore:      mockWalletsStore,
				transactionsStore: mockTransactionsStore,
				tx:                mockTx,
				xrClient:          mockXRClient,
				producer:          mockTxProducer,
				metrics:           getTestMetrics(),
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
