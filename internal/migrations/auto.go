package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func main() {

	pool, err := pgxpool.New(context.Background(), os.Getenv("DSN"))
	if err != nil {
		panic(err)
	}

	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS IPS (
			address TEXT PRIMARY KEY,
			count INT 
		)
	`)

	if err != nil {
		log.Error().Err(err).Msg("Automigration failure")
	}
}
