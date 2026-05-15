package postgres

import (
	"context"
	"fmt"
	"log"

	// _ "github.com/jackc/puddle/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func Connect(databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping PostgreSQL: %w", err)
	}

	log.Println("PostgreSQL connected")
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
	log.Println("PostgreSQL disconnected")
}
