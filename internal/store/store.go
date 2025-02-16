package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // functions from this package are not used
	"github.com/romanpitatelev/wallets-service/configs"
	ip "github.com/romanpitatelev/wallets-service/internal/model"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations
var migrations embed.FS

type VisitorStore struct {
	pool *pgxpool.Pool
	dsn  string
}

func New(ctx context.Context, conf *configs.Config) (*VisitorStore, error) {
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

	return &VisitorStore{
		pool: pool,
		dsn:  dsn,
	}, nil
}

func (v *VisitorStore) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", v.dsn)
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

func (v *VisitorStore) Add(ctx context.Context, ipAddress string) error {
	var ipRecord ip.IP

	err := v.pool.QueryRow(ctx,
		`INSERT INTO ips (ipaddress, count) 
		VALUES ($1, $2) 
		ON CONFLICT (ipaddress) 
		DO UPDATE SET count = ips.count + 1 
		RETURNING ipaddress, count`,
		ipAddress, 1).Scan(&ipRecord.Address, &ipRecord.Count)
	if err != nil {
		return fmt.Errorf("failed to add ip: %w", err)
	}

	return nil
}

func (v *VisitorStore) GetVisitsAll(ctx context.Context) (map[string]int, error) {
	rows, err := v.pool.Query(ctx, "SELECT ipaddress, count FROM ips")
	if err != nil {
		return nil, fmt.Errorf("failed to query visits: %w", err)
	}
	defer rows.Close()

	visits := make(map[string]int)

	for rows.Next() {
		var ipRecord ip.IP

		err = rows.Scan(&ipRecord.Address, &ipRecord.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to iterate visits: %w", err)
		}

		visits[ipRecord.Address] = ipRecord.Count
	}

	return visits, nil
}
