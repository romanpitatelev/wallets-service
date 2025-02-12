package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/romanpitatelev/wallets-service/internal/ip"
)

type VisitorStore struct {
	pool *pgxpool.Pool
}

func NewVisitorStore(pool *pgxpool.Pool) *VisitorStore {
	return &VisitorStore{
		pool: pool,
	}
}

func (v *VisitorStore) Add(ipAddress string) {
	ctx := context.Background()

	var ipRecord ip.IP

	err := v.pool.QueryRow(ctx,
		`INSERT INTO ips (address, count) 
		VALUES ($1, $2) 
		ON CONFLICT (address) 
		DO UPDATE SET count = ips.count + 1 
		RETURNING address, count`,
		ipAddress, 1).Scan(&ipRecord.Address, &ipRecord.Count)
	if err != nil {
		panic(err)
	}
}

func (v *VisitorStore) GetVisitsAll() map[string]int {
	ctx := context.Background()
	rows, err := v.pool.Query(ctx, "SELECT address, count FROM ips")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	visits := make(map[string]int)

	for rows.Next() {
		var ipRecord ip.IP
		err = rows.Scan(&ipRecord.Address, &ipRecord.Count)
		if err != nil {
			panic(err)
		}

		visits[ipRecord.Address] = ipRecord.Count
	}

	return visits
}
