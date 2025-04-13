package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/romanpitatelev/wallets-service/internal/entity"
	"github.com/rs/zerolog/log"
)

func (d *DataStore) Deposit(ctx context.Context, transaction entity.Transaction, userID entity.UserID, rate float64) error {
	tx := d.getTXFromCtx(ctx)

	query := `
UPDATE wallets
SET balance = balance + $3::numeric * $4::numeric, updated_at = NOW() 
WHERE TRUE
	AND wallet_id = $1 
	AND user_id = $2 
	AND active = true`

	result, err := tx.Exec(ctx, query, transaction.ToWalletID, userID, transaction.Amount, rate)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance info: %w", err)
	}

	if result.RowsAffected() == 0 {
		return entity.ErrWalletNotFound
	}

	transaction.Type = "deposit"

	if err := d.storeTxIntoTable(ctx, transaction, tx); err != nil {
		return fmt.Errorf("failed to store transaction into database: %w", err)
	}

	return nil
}

func (d *DataStore) Withdraw(ctx context.Context, transaction entity.Transaction, userID entity.UserID, rate float64) error {
	tx := d.getTXFromCtx(ctx)

	query := `
UPDATE wallets
SET balance = balance - $3::numeric * $4::numeric, updated_at = NOW()
WHERE TRUE 
	AND wallet_id = $1 
	AND user_id = $2 
	AND active = true`

	result, err := tx.Exec(ctx, query, transaction.FromWalletID, userID, transaction.Amount, rate)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance info: %w", err)
	}

	if result.RowsAffected() == 0 {
		return entity.ErrWalletNotFound
	}

	transaction.Type = "withdraw"

	if err := d.storeTxIntoTable(ctx, transaction, tx); err != nil {
		return fmt.Errorf("failed to store transaction into database: %w", err)
	}

	return nil
}

func (d *DataStore) Transfer(ctx context.Context, transaction entity.Transaction, userID entity.UserID, rate float64) error {
	tx := d.getTXFromCtx(ctx)

	queryFrom := `
UPDATE wallets
SET balance = balance - $3::numeric, updated_at = NOW()
WHERE TRUE 
	AND wallet_id = $1 
	AND user_id = $2 
	AND active = true`

	resultFrom, err := tx.Exec(ctx, queryFrom, transaction.FromWalletID, userID, transaction.Amount)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance info: %w", err)
	}

	if resultFrom.RowsAffected() == 0 {
		return entity.ErrWalletNotFound
	}

	queryTo := `
UPDATE wallets
SET balance = balance + $3::numeric * $4::numeric, updated_at = NOW()
WHERE TRUE 
	AND wallet_id = $1 
	AND user_id = $2 
	AND active = true`

	resultTo, err := tx.Exec(ctx, queryTo, transaction.ToWalletID, userID, transaction.Amount, rate)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance info: %w", err)
	}

	if resultTo.RowsAffected() == 0 {
		return entity.ErrWalletNotFound
	}

	transaction.Type = "transfer"

	if err := d.storeTxIntoTable(ctx, transaction, tx); err != nil {
		return fmt.Errorf("failed to store transaction into database: %w", err)
	}

	return nil
}

//nolint:lll
func (d *DataStore) GetTransactions(ctx context.Context, request entity.GetWalletsRequest, walletID entity.WalletID, userID entity.UserID) ([]entity.Transaction, error) {
	_, err := d.GetWallet(ctx, walletID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract wallet: %w", err)
	}

	var (
		transactionsAll []entity.Transaction
		rows            pgx.Rows
	)

	query, args := d.GetTransactionsQuery(request, walletID)

	if rows, err = d.pool.Query(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("error getting all the transactions: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var transaction entity.Transaction

		err = rows.Scan(
			&transaction.ID,
			&transaction.Type,
			&transaction.ToWalletID,
			&transaction.FromWalletID,
			&transaction.Amount,
			&transaction.Currency,
			&transaction.CommittedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error when scanning transactions: %w", err)
		}

		transactionsAll = append(transactionsAll, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	if len(transactionsAll) == 0 {
		return []entity.Transaction{}, nil
	}

	return transactionsAll, nil
}

func (d *DataStore) GetTransactionsQuery(request entity.GetWalletsRequest, walletID entity.WalletID) (string, []any) {
	var (
		sb              strings.Builder
		args            []any
		validSortParams = map[string]string{
			"transaction_type": "transaction_type",
			"currency":         "currency",
		}
	)

	sb.WriteString(`SELECT id, transaction_type, to_wallet_id, from_wallet_id, amount, currency, committed_at
						FROM transactions
						WHERE`)

	args = append(args, walletID)
	sb.WriteString(fmt.Sprintf(` to_wallet_id = $%d`, len(args)))
	args = append(args, walletID)
	sb.WriteString(fmt.Sprintf(` OR from_wallet_id = $%d`, len(args)))

	if request.Filter != "" {
		args = append(args, "%"+request.Filter+"%")
		sb.WriteString(fmt.Sprintf(` AND concat_ws('', id, transaction_type, amount, currency, committed_at) ILIKE $%d`, len(args)))
	}

	sorting, ok := validSortParams[request.Sorting]
	if !ok {
		sorting = "transaction_type"
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

func (d *DataStore) storeTxIntoTable(ctx context.Context, transaction entity.Transaction, tx transaction) error {
	transaction.CommittedAt = time.Now()

	query := `
INSERT INTO transactions (id, transaction_type, to_wallet_id, from_wallet_id, amount, currency, committed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

	args := []any{
		uuid.New(),
		transaction.Type,
		nil,
		nil,
		transaction.Amount,
		transaction.Currency,
		transaction.CommittedAt,
	}

	if transaction.ToWalletID != nil {
		args[2] = transaction.ToWalletID
	}

	if transaction.FromWalletID != nil {
		args[3] = transaction.FromWalletID
	}

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			log.Error().Err(err).Msg("wallet not found: foreign key violation")

			return entity.ErrWalletNotFound
		}

		return fmt.Errorf("failed to save transaction history in database: %w", err)
	}

	return nil
}
