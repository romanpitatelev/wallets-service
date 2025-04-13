package walletsrepo

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/romanpitatelev/wallets-service/internal/reporsitory/store"
	"strings"
	"time"
)

type db interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
	GetTXFromCtx(ctx context.Context) store.Transaction
}

type Repo struct {
	db db
}

func New(db db) *Repo {
	return &Repo{
		db: db,
	}
}

func (d *Repo) CreateWallet(ctx context.Context, wallet entity.Wallet, userID entity.UserID) (entity.Wallet, error) {
	query := `
INSERT INTO wallets (wallet_id, user_id, wallet_name, currency)
VALUES ($1, $2, $3, $4)
RETURNING wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, active`

	row := d.db.QueryRow(ctx, query,
		wallet.WalletID,
		userID,
		wallet.WalletName,
		wallet.Currency,
	)

	var createdWallet entity.Wallet

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
		return entity.Wallet{}, fmt.Errorf("failed to create wallet: %w", err)
	}

	return createdWallet, nil
}

func (d *Repo) GetWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) (entity.Wallet, error) {
	var wallet entity.Wallet

	query := `
SELECT wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, active
FROM wallets
WHERE TRUE 
	AND wallet_id = $1 
	AND user_id = $2 
	AND deleted_at IS NULL`

	var db store.Transaction

	db = d.db.GetTXFromCtx(ctx)

	if db == nil {
		db = d.db
	} else {
		query += ` FOR UPDATE`
	}

	err := db.QueryRow(ctx, query, walletID, userID).Scan(
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
			return entity.Wallet{}, entity.ErrWalletNotFound
		}

		return entity.Wallet{}, fmt.Errorf("failed to get wallet info: %w", err)
	}

	return wallet, nil
}

//nolint:lll
func (d *Repo) UpdateWallet(ctx context.Context, walletID entity.WalletID, newInfoWallet entity.WalletUpdate, rate float64, userID entity.UserID) (entity.Wallet, error) {
	tx := d.db.GetTXFromCtx(ctx)

	query := `
UPDATE wallets
SET wallet_name = $1, currency = $2, balance = $3 * balance, updated_at = $4
WHERE TRUE 
	AND wallet_id = $5 
	AND user_id = $6 
	AND deleted_at IS NULL
RETURNING wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, deleted_at, active`

	updatedAt := time.Now()

	row := tx.QueryRow(ctx, query,
		newInfoWallet.WalletName,
		strings.ToUpper(newInfoWallet.Currency),
		rate,
		updatedAt,
		walletID,
		userID,
	)

	var wallet entity.Wallet

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
			return entity.Wallet{}, entity.ErrWalletNotFound
		}

		return entity.Wallet{}, fmt.Errorf("failed to get wallet info: %w", err)
	}

	return wallet, nil
}

func (d *Repo) DeleteWallet(ctx context.Context, walletID entity.WalletID, userID entity.UserID) error {
	currentWallet, err := d.GetWallet(ctx, walletID, userID)
	if err != nil {
		if errors.Is(err, entity.ErrWalletNotFound) {
			return entity.ErrWalletNotFound
		}

		return fmt.Errorf("failed to fetch current wallet in DeleteWallet() function: %w", err)
	}

	if currentWallet.Balance != 0.0 {
		return entity.ErrNonZeroBalanceWallet
	}

	query := `
UPDATE wallets
SET deleted_at = NOW(), active = false
WHERE TRUE 
	AND wallet_id = $1 
	AND user_id = $2
	AND deleted_at IS NULL 
	AND active = true`

	_, err = d.db.Exec(ctx, query, walletID, userID)
	if err != nil {
		return fmt.Errorf("error deleting wallet %s: %w", uuid.UUID(walletID).String(), err)
	}

	return nil
}

func (d *Repo) GetWallets(ctx context.Context, request entity.GetWalletsRequest, userID entity.UserID) ([]entity.Wallet, error) {
	var (
		walletsAll []entity.Wallet
		rows       pgx.Rows
		err        error
	)

	query, args := d.GetWalletsQuery(request, userID)

	if rows, err = d.db.Query(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("error getting all wallets info: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var wallet entity.Wallet

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
		return []entity.Wallet{}, nil
	}

	return walletsAll, nil
}

func (d *Repo) GetWalletsQuery(request entity.GetWalletsRequest, userID entity.UserID) (string, []any) {
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

func (d *Repo) ArchiveStaleWallets(ctx context.Context, checkPeriod time.Duration) error {
	query := fmt.Sprintf(`UPDATE wallets
				SET active = false
				WHERE balance = 0
					AND active = true 
					AND updated_at < NOW() - INTERVAL '%d hours'`, int(checkPeriod.Hours()))

	_, err := d.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("error archiving wallet: %w", err)
	}

	return nil
}
