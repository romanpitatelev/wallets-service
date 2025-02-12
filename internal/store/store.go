package store

import (
	"context"
	"fmt"
	"github.com/romanpitatelev/wallets-service/internal/configs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/romanpitatelev/wallets-service/internal/model"
)

type VisitorStore struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, conf *configs.Config) (*VisitorStore, error) {
	pool, err := pgxpool.New(ctx, conf.DB.DSN)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	return &VisitorStore{
		pool: pool,
	}, nil
}

func (v *VisitorStore) Add(ctx context.Context, ipAddress string) error {
	var ipRecord model.IP

	err := v.pool.QueryRow(ctx,
		`INSERT INTO ips (address, count) 
		VALUES ($1, $2) 
		ON CONFLICT (address) 
		DO UPDATE SET count = ips.count + 1 
		RETURNING address, count`,
		ipAddress, 1).Scan(&ipRecord.Address, &ipRecord.Count)
	if err != nil {
		return fmt.Errorf("failed to add ip: %w", err)
	}

	return nil
}

func (v *VisitorStore) GetVisitsAll(ctx context.Context) (map[string]int, error) {
	rows, err := v.pool.Query(ctx, `SELECT address, count FROM ips`)
	if err != nil {
		return nil, fmt.Errorf("failed to query visits: %w", err)
	}
	defer rows.Close()

	visits := make(map[string]int)

	for rows.Next() {
		var ipRecord model.IP

		if err = rows.Scan(&ipRecord.Address, &ipRecord.Count); err != nil {
			return nil, fmt.Errorf("failed to scan visits: %w", err)
		}

		visits[ipRecord.Address] = ipRecord.Count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate visits: %w", err)
	}

	return visits, nil
}
