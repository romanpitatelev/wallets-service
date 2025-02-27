package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
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

	return nil
}

func (d *DataStore) UpsertUser(ctx context.Context, users models.User) error {
	query := `INSERT INTO users (userid, deleted)
		VALUES ($1, $2)
		ON CONFLICT (userid) 
		DO UPDATE SET deleted = excluded.deleted
		RETURNING userid, deleted`

	_, err := d.pool.Exec(ctx, query, users.UserID, users.Deleted)
	if err != nil {
		return fmt.Errorf("failed to upsert users: %w", err)
	}

	return nil
}

func (d *DataStore) CreateWallet(ctx context.Context, wallet models.Wallet) error {
	query := `INSERT INTO wallets (walletId, walletName, balance, currency, createdAt, updatedAt, deletedAt)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING walletId, walletName, balance, currency, createdAt, updatedAt, deletedAt`

	row := d.pool.QueryRow(ctx, query,
		wallet.WalletID,
		wallet.WalletName,
		wallet.Balance,
		wallet.Currency,
		wallet.CreatedAt,
		wallet.UpdatedAt,
		wallet.DeletedAt,
	)
	// TODO do properly
	if err := row.Scan(&wallet); err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	return nil
}

func (d *DataStore) GetWallet(ctx context.Context, walletID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet

	query := `SELECT walletId, walletName, balance, currency, createdAt, updatedAt, deletedAt
		FROM wallets
		WHERE walletId = $1 AND deletedAt IS NULL`

	err := d.pool.QueryRow(ctx, query, walletID).Scan(
		&wallet.WalletID,
		&wallet.WalletName,
		&wallet.Balance,
		&wallet.Currency,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
		&wallet.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to get wallet %s by id: %w", walletID.String(), err)
		}

		return nil, fmt.Errorf("failed to get wallet info: %w", err)
	}

	return &wallet, nil
}

func (d *DataStore) UpdateWallet(ctx context.Context, wallet models.Wallet) error {
	var exists bool

	existQuery := `SELECT EXISTS
		(SELECT 1 FROM wallets 
		WHERE walletid = $1 
		AND deletedAt IS NULL)`

	err := d.pool.QueryRow(ctx, existQuery, wallet.WalletID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if wallet exists: %w", err)
	}

	if !exists {
		return models.ErrWalletNotFound
	}

	query := `UPDATE wallets
		SET walletName = $2, balance = $3, currency = $4, updatedAt = $5
		WHERE walletId = $1 AND deletedAt IS NULL`

	result, err := d.pool.Exec(ctx, query,
		wallet.WalletID,
		wallet.WalletName,
		wallet.Balance,
		wallet.Currency,
		wallet.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error updating wallet %s: %w", wallet.WalletID.String(), err)
	}

	if result.RowsAffected() == 0 {
		return models.ErrWalletUpToDate
	}

	return nil
}

func (d *DataStore) DeleteWallet(ctx context.Context, walletID uuid.UUID) error {
	query := `UPDATE wallets
		SET deletedAt = NOW()
		WHERE walletId = $1`

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
	query := `SELECT walletId, walletName, balance, currency, createdAt, updatedAt, deletedAt
	FROM wallets
	WHERE deletedAt IS NULL`

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
			&wallet.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error when scanning wallet: %w", err)
		}

		walletsAll = append(walletsAll, wallet)
	}

	return walletsAll, nil
}
