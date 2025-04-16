package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/romanpitatelev/wallets-service/internal/entity"
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

func New(ctx context.Context, cfg Config) (*DataStore, error) {
	pool, err := pgxpool.New(ctx, cfg.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("connected to database")

	return &DataStore{
		pool: pool,
		dsn:  cfg.Dsn,
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
		log.Info().Str("file", file.Name()).Msg("found migration file")
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

func (d *DataStore) UpsertUser(ctx context.Context, users entity.User) error {
	query := `
INSERT INTO users (user_id, deleted_at)
VALUES ($1, $2)
ON CONFLICT (user_id) 
DO UPDATE 
SET deleted_at = excluded.deleted_at`

	_, err := d.pool.Exec(ctx, query, users.UserID, users.DeletedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert users: %w", err)
	}

	return nil
}

func (d *DataStore) Truncate(ctx context.Context, tables ...string) error {
	for _, table := range tables {
		if _, err := d.pool.Exec(ctx, `DELETE FROM `+table); err != nil {
			return fmt.Errorf("error truncating wallet %s: %w", table, err)
		}
	}

	return nil
}

func (d *DataStore) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	cmdTag, err := d.pool.Exec(ctx, query, args...)
	if err != nil {
		return pgconn.CommandTag{}, fmt.Errorf("error executing query %s: %w", query, err)
	}

	return cmdTag, nil
}

func (d *DataStore) DoWithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

type txtCtxKey string

//nolint:gochecknoglobals
var ctxKey txtCtxKey = "tx"

func (d *DataStore) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	res, err := d.pool.Query(ctx, sql, arguments...)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", sql, err)
	}

	return res, nil
}

func (d *DataStore) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	return d.pool.QueryRow(ctx, sql, arguments...)
}

type Transaction interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

func (d *DataStore) GetTXFromCtx(ctx context.Context) Transaction {
	tx, ok := ctx.Value(ctxKey).(pgx.Tx)
	if !ok {
		return d.pool
	}

	return tx
}

func (d *DataStore) storeTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, ctxKey, tx)
}
