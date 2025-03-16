package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // functions from this package are not used
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations
var migrations embed.FS

type DataStore struct {
	pool *pgxpool.Pool
	dsn  string
}

type Config struct {
	Dsn string
}

func New(ctx context.Context, conf Config) (*DataStore, error) {
	pool, err := pgxpool.New(ctx, conf.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("connected to database")

	return &DataStore{
		pool: pool,
		dsn:  conf.Dsn,
	}, nil
}

func (d *DataStore) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", d.dsn)
	if err != nil {
		return fmt.Errorf("failed to open sql: %w", err)
	}

	defer func() {
		err := conn.Close()
		if err != nil {
			log.Error().Msg("failed to close database")
		}
	}()

	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		log.Debug().Str("file", file.Name()).Msg("found migration file")
	}

	assetDir := func() func(string) ([]string, error) {
		return func(path string) ([]string, error) {
			dirEntry, err := migrations.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("migrations reading failed: %w", err)
			}

			entries := make([]string, 0)
			for _, e := range dirEntry {
				entries = append(entries, e.Name())
			}

			return entries, nil
		}
	}()

	asset := migrate.AssetMigrationSource{
		Asset:    migrations.ReadFile,
		AssetDir: assetDir,
		Dir:      "migrations",
	}

	_, err = migrate.Exec(conn, "postgres", asset, direction)
	if err != nil {
		return fmt.Errorf("failed to count the number of migrations: %w", err)
	}

	return nil
}

func (d *DataStore) UpsertUser(ctx context.Context, users models.User) error {
	query := `INSERT INTO users (user_id, deleted_at)
		VALUES ($1, $2)
		ON CONFLICT (user_id) 
		DO UPDATE SET deleted_at = excluded.deleted_at`

	_, err := d.pool.Exec(ctx, query, users.UserID, users.DeletedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert users: %w", err)
	}

	return nil
}

func (d *DataStore) CreateWallet(ctx context.Context, wallet models.Wallet, userID uuid.UUID) (models.Wallet, error) {
	query := `INSERT INTO wallets (wallet_id, user_id, wallet_name, currency)
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
		log.Error().Err(err).Msg("Failed to insert wallet into database")

		return models.Wallet{}, fmt.Errorf("failed to create wallet: %w", err)
	}

	return createdWallet, nil
}

func (d *DataStore) GetWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) (models.Wallet, error) {
	var wallet models.Wallet

	query := `SELECT wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, active
				FROM wallets
				WHERE wallet_id = $1 AND user_id = $2 AND deleted_at IS NULL`

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
func (d *DataStore) UpdateWallet(ctx context.Context, walletID uuid.UUID, newInfoWallet models.WalletUpdate, rate float64, userID uuid.UUID) (models.Wallet, error) {
	currentWallet, err := d.GetWallet(ctx, walletID, userID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to fetch current wallet: %w", err)
	}

	if newInfoWallet.WalletName == "" {
		newInfoWallet.WalletName = currentWallet.WalletName
	}

	if newInfoWallet.Currency == "" {
		newInfoWallet.Currency = currentWallet.Currency
	}

	query := `UPDATE wallets
		SET wallet_name = $1, currency = $2, balance = $3 * balance, updated_at = $4
		WHERE wallet_id = $5 AND user_id = $6 AND deleted_at IS NULL
		RETURNING wallet_id, user_id, wallet_name, balance, currency, created_at, updated_at, deleted_at, active`

	updatedAt := time.Now()

	row := d.pool.QueryRow(ctx, query,
		newInfoWallet.WalletName,
		strings.ToUpper(newInfoWallet.Currency),
		rate,
		updatedAt,
		walletID,
		userID,
	)

	var wallet models.Wallet

	err = row.Scan(
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

func (d *DataStore) DeleteWallet(ctx context.Context, walletID uuid.UUID, userID uuid.UUID) error {
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

	query := `UPDATE wallets
				SET deleted_at = NOW(), active = false
				WHERE wallet_id = $1 
					AND user_id = $2
					AND deleted_at IS NULL 
					AND active = true`

	_, err = d.pool.Exec(ctx, query, walletID, userID)
	if err != nil {
		return fmt.Errorf("error deleting wallet %s: %w", walletID.String(), err)
	}

	return nil
}

func (d *DataStore) GetWallets(ctx context.Context, request models.GetWalletsRequest, userID uuid.UUID) ([]models.Wallet, error) {
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

func (d *DataStore) GetWalletsQuery(request models.GetWalletsRequest, userID uuid.UUID) (string, []any) {
	var (
		sb              strings.Builder
		args            []any
		validSortParams = map[string]string{
			"wallet_id":   "wallet_id",
			"wallet_name": "wallet_name",
			"currency":    "currency",
			"balance":     "balance",
			"created_at":  "created_at",
			"updated_at":  "updated_at",
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
		sorting = "wallet_id"
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

func (d *DataStore) Truncate(ctx context.Context, tables ...string) error {
	for _, table := range tables {
		if _, err := d.pool.Exec(ctx, `DELETE FROM `+table); err != nil {
			return fmt.Errorf("error truncating wallet %s: %w", table, err)
		}
	}

	return nil
}

func (d *DataStore) ArchiveStaleWallets(ctx context.Context, checkPeriod time.Duration) error {
	query := fmt.Sprintf(`UPDATE wallets
				SET active = false
				WHERE balance = 0 
				AND updated_at < NOW() - INTERVAL '%d hours'`, int(checkPeriod.Hours()))

	_, err := d.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("error archiving wallet: %w", err)
	}

	return nil
}

func (d *DataStore) Exec(ctx context.Context, query string, args ...any) error {
	if _, err := d.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("error executing query %s: %w", query, err)
	}

	return nil
}
