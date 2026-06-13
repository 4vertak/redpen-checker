package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/redpen?sslmode=disable"
	}
	var err error
	Pool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к базе: %w", err)
	}
	if err = Pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("пинг базы не прошёл: %w", err)
	}
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}