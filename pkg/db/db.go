package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/romanpitatelev/wallets-service/configs"
)

type Db struct {
	Pool *pgxpool.Pool
}

func NewDb(conf *configs.Config) *Db {
	pool, err := pgxpool.New(context.Background(), conf.Db.Dsn)
	if err != nil {
		panic(err)
	}

	return &Db{Pool: pool}
}
