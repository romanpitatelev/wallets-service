package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib" // functions from this package are not used
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
)

func (d *DataStore) CreateWallet(ctx context.Context, wallet models.Wallet, userID models.UserID) (models.Wallet, error) {
	query := `
INSERT INTO wallets (wallet_id, user_id, wallet_name, currency)
VALUES ($1, $2, $3, $4)
RETURNING wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, active`

	row := d.pool.QueryRow(ctx, query,
		wallet.WalletID,
		userID,
		wallet.WalletName,
		wallet.Currency,
	)

	var createdWallet models.Wallet

	err := row.Scan(
		&createdWallet.WalletID,
		&createdWallet.UserID,
		&createdWallet.WalletName,
		&createdWallet.Balance,
		&createdWallet.Currency,
		&createdWallet.CreatedAt,
		&createdWallet.UpdatedAt,
		&createdWallet.Active,
	)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to create wallet: %w", err)
	}

	return createdWallet, nil
}

type querier interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

//nolint:ineffassign,wastedassign
func (d *DataStore) GetWallet(ctx context.Context, walletID models.WalletID, userID models.UserID) (models.Wallet, error) {
	var wallet models.Wallet

	query := `
SELECT wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, active
FROM wallets
WHERE TRUE 
	AND wallet_id = $1 
	AND user_id = $2 
	AND deleted_at IS NULL`

	var db querier

	db = d.getTXFromCtx(ctx)

	if db == nil {
		db = d.pool
	} else {
		query += ` FOR UPDATE`
	}

	err := d.pool.QueryRow(ctx, query, walletID, userID).Scan(
		&wallet.WalletID,
		&wallet.UserID,
		&wallet.WalletName,
		&wallet.Balance,
		&wallet.Currency,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
		&wallet.Active,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Wallet{}, models.ErrWalletNotFound
		}

		return models.Wallet{}, fmt.Errorf("failed to get wallet info: %w", err)
	}

	return wallet, nil
}

//nolint:lll
func (d *DataStore) UpdateWallet(ctx context.Context, walletID models.WalletID, newInfoWallet models.WalletUpdate, rate float64, userID models.UserID) (models.Wallet, error) {
	query := `
UPDATE wallets
SET wallet_name = $1, currency = $2, balance = $3 * balance, updated_at = $4
WHERE TRUE 
	AND wallet_id = $5 
	AND user_id = $6 
	AND deleted_at IS NULL
RETURNING wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, deleted_at, active`

	updatedAt := time.Now()

	// TODO review and rework
	tx := d.getTXFromCtx(ctx)

	row := tx.QueryRow(ctx, query,
		newInfoWallet.WalletName,
		strings.ToUpper(newInfoWallet.Currency),
		rate,
		updatedAt,
		walletID,
		userID,
	)

	var wallet models.Wallet

	err := row.Scan(
		&wallet.WalletID,
		&wallet.UserID,
		&wallet.WalletName,
		&wallet.Balance,
		&wallet.Currency,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
		&wallet.DeletedAt,
		&wallet.Active,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Wallet{}, models.ErrWalletNotFound
		}

		return models.Wallet{}, fmt.Errorf("failed to get wallet info: %w", err)
	}

	return wallet, nil
}

func (d *DataStore) DeleteWallet(ctx context.Context, walletID models.WalletID, userID models.UserID) error {
	currentWallet, err := d.GetWallet(ctx, walletID, userID)
	if err != nil {
		if errors.Is(err, models.ErrWalletNotFound) {
			return models.ErrWalletNotFound
		}

		return fmt.Errorf("failed to fetch current wallet in DeleteWallet() function: %w", err)
	}

	if currentWallet.Balance != 0.0 {
		return models.ErrNonZeroBalanceWallet
	}

	query := `
UPDATE wallets
SET deleted_at = NOW(), active = false
WHERE TRUE 
	AND wallet_id = $1 
	AND user_id = $2
	AND deleted_at IS NULL 
	AND active = true`

	_, err = d.pool.Exec(ctx, query, walletID, userID)
	if err != nil {
		return fmt.Errorf("error deleting wallet %s: %w", uuid.UUID(walletID).String(), err)
	}

	return nil
}

func (d *DataStore) GetWallets(ctx context.Context, request models.GetWalletsRequest, userID models.UserID) ([]models.Wallet, error) {
	var (
		walletsAll []models.Wallet
		rows       pgx.Rows
		err        error
	)

	query, args := d.GetWalletsQuery(request, userID)

	if rows, err = d.pool.Query(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("error getting all wallets info: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var wallet models.Wallet

		err = rows.Scan(
			&wallet.WalletID,
			&wallet.UserID,
			&wallet.WalletName,
			&wallet.Balance,
			&wallet.Currency,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
			&wallet.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("error when scanning wallet: %w", err)
		}

		walletsAll = append(walletsAll, wallet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	if len(walletsAll) == 0 {
		return []models.Wallet{}, nil
	}

	return walletsAll, nil
}

func (d *DataStore) GetWalletsQuery(request models.GetWalletsRequest, userID models.UserID) (string, []any) {
	var (
		sb              strings.Builder
		args            []any
		validSortParams = map[string]string{
			"wallet_name": "wallet_name",
			"currency":    "currency",
		}
	)

	sb.WriteString(`SELECT wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, active
					FROM wallets
					WHERE deleted_at IS NULL
						AND active = true`)

	args = append(args, userID)
	sb.WriteString(fmt.Sprintf(` AND user_id = $%d`, len(args)))

	if request.Filter != "" {
		args = append(args, "%"+request.Filter+"%")
		sb.WriteString(fmt.Sprintf(` AND concat_ws('', wallet_id, wallet_name, currency, balance, created_at, updated_at) ILIKE $%d`, len(args)))
	}

	sorting, ok := validSortParams[request.Sorting]
	if !ok {
		sorting = "currency"
	}

	sb.WriteString(" ORDER BY " + sorting)

	if request.Descending {
		sb.WriteString(" DESC")
	}

	args = append(args, request.Limit)

	sb.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))

	if request.Offset > 0 {
		args = append(args, request.Offset)
		sb.WriteString(fmt.Sprintf(" OFFSET $%d", len(args)))
	}

	return sb.String(), args
}

// TODO move to db.go

func (d *DataStore) DoWithTx(ctx context.Context, txName string, fn func(ctx context.Context) error) error {
	started := time.Now()
	defer func() {
		d.metrics.dbResponseDuration.WithLabelValues(txName).Observe(time.Since(started).Seconds())
	}()

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	ctx = d.storeTx(ctx, tx)

	defer func() {
		if err = tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Warn().Err(err).Msg("failed to rollback transaction")
		}
	}()

	if err := fn(ctx); err != nil {
		return fmt.Errorf("error in fn(ctx): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
