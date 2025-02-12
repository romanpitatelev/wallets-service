package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

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
		log.Error().Msg("Automigration failure")
	}
}
