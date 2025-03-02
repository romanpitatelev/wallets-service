package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // functions from this package are not used
	"github.com/romanpitatelev/wallets-service/configs"
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

func New(ctx context.Context, conf *configs.Config) (*DataStore, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		conf.PostgresHost,
		conf.PostgresUser,
		conf.PostgresPassword,
		conf.PostgresDatabase,
		conf.PostgresPort,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Debug().Msg("connection to db successful")

	return &DataStore{
		pool: pool,
		dsn:  dsn,
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

	log.Debug().Msg("Migrations applied successfully")

	return nil
}

func (d *DataStore) UpsertUser(ctx context.Context, users models.User) error {
	query := `INSERT INTO users (userid, deleted)
		VALUES ($1, $2)
		ON CONFLICT (userid) 
		DO UPDATE SET deleted_at = excluded.deleted_at
`

	_, err := d.pool.Exec(ctx, query, users.UserID, users.DeletedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert users: %w", err)
	}

	return nil
}

func (d *DataStore) CreateWallet(ctx context.Context, wallet models.Wallet) (models.Wallet, error) {
	query := `INSERT INTO wallets (wallet_id, wallet_name, currency)
		VALUES ($1, $2, $3)
		RETURNING wallet_id, wallet_name, balance, currency, created_at, updated_at`

	log.Debug().Interface("wallet", wallet).Msg("Inserting wallet into database")

	row := d.pool.QueryRow(ctx, query,
		wallet.WalletID,
		wallet.WalletName,
		wallet.Currency,
	)

	var createdWallet models.Wallet

	err := row.Scan(
		&createdWallet.WalletID,
		&createdWallet.WalletName,
		&createdWallet.Balance,
		&createdWallet.Currency,
		&createdWallet.CreatedAt,
		&createdWallet.UpdatedAt,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert wallet into database")

		return models.Wallet{}, fmt.Errorf("failed to create wallet: %w", err)
	}

	log.Debug().Interface("createdWallet", createdWallet).Msg("Wallet created successfully")

	return createdWallet, nil
}

func (d *DataStore) GetWallet(ctx context.Context, walletID uuid.UUID) (models.Wallet, error) {
	var wallet models.Wallet

	query := `SELECT wallet_id, wallet_name, balance, currency, created_at, updated_at
		FROM wallets
		WHERE wallet_id = $1
		  AND deleted_at IS NULL`

	err := d.pool.QueryRow(ctx, query, walletID).Scan(
		&wallet.WalletID,
		&wallet.WalletName,
		&wallet.Balance,
		&wallet.Currency,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Wallet{}, models.ErrWalletNotFound
		}

		return models.Wallet{}, fmt.Errorf("failed to get wallet info: %w", err)
	}

	return wallet, nil
}

func (d *DataStore) UpdateWallet(ctx context.Context, walletID uuid.UUID, newInfoWallet models.WalletUpdate) (models.Wallet, error) {
	currentWallet, err := d.GetWallet(ctx, walletID)
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
		SET wallet_name = $1, currency = $2, updated_at = $3
		WHERE wallet_id = $4 AND deleted_at IS NULL
		RETURNING wallet_id, wallet_name, balance, currency, created_at, updated_at`

	updatedAt := time.Now()

	row := d.pool.QueryRow(ctx, query,
		newInfoWallet.WalletName,
		newInfoWallet.Currency,
		updatedAt,
		walletID,
	)

	var wallet models.Wallet

	err = row.Scan(
		&wallet.WalletID,
		&wallet.WalletName,
		&wallet.Balance,
		&wallet.Currency,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Wallet{}, models.ErrWalletNotFound
		}

		return models.Wallet{}, fmt.Errorf("failed to get wallet info: %w", err)
	}

	return wallet, nil
}

func (d *DataStore) DeleteWallet(ctx context.Context, walletID uuid.UUID) error {
	query := `UPDATE wallets
		SET deleted_at = NOW()
		WHERE wallet_id = $1 AND deleted_at IS NULL`

	result, err := d.pool.Exec(ctx, query, walletID)
	if err != nil {
		return fmt.Errorf("error deleting wallet %s: %w", walletID.String(), err)
	}

	if result.RowsAffected() == 0 {
		return models.ErrWalletNotFound
	}

	return nil
}

func (d *DataStore) GetWallets(ctx context.Context) ([]models.Wallet, error) {
	query := `
SELECT wallet_id, wallet_name, balance, currency, created_at, updated_at
FROM wallets 
WHERE deleted_at IS NULL`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error getting all wallets info: %w", err)
	}
	defer rows.Close()

	var walletsAll []models.Wallet

	for rows.Next() {
		var wallet models.Wallet

		err = rows.Scan(
			&wallet.WalletID,
			&wallet.WalletName,
			&wallet.Balance,
			&wallet.Currency,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error when scanning wallet: %w", err)
		}

		walletsAll = append(walletsAll, wallet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	return walletsAll, nil
}

func (d *DataStore) Truncate(ctx context.Context, tables ...string) error {
	for _, table := range tables {
		if _, err := d.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s`, table)); err != nil {
			return fmt.Errorf("error truncating wallet %s: %w", table, err)
		}
	}

	return nil
}
