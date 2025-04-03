package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/romanpitatelev/wallets-service/internal/models"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations
var migrations embed.FS

type DataStore struct {
	pool    *pgxpool.Pool
	dsn     string
	metrics *metrics
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
		pool:    pool,
		dsn:     conf.Dsn,
		metrics: newMetrics(),
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

func (d *DataStore) UpsertUser(ctx context.Context, users models.User) error {
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

func (d *DataStore) ArchiveStaleWallets(ctx context.Context, checkPeriod time.Duration) error {
	query := fmt.Sprintf(`UPDATE wallets
				SET active = false
				WHERE balance = 0
					AND active = true 
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

type txtCtxKey string

//nolint:gochecknoglobals
var ctxKey txtCtxKey = "tx"

func (d *DataStore) storeTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, ctxKey, tx)
}

type transaction interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

func (d *DataStore) getTXFromCtx(ctx context.Context) transaction {
	tx, ok := ctx.Value(ctxKey).(pgx.Tx)
	if !ok {
		return d.pool
	}

	return tx
}
