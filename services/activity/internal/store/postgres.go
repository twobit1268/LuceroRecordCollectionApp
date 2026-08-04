package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	s := &PostgresStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS activity_entries (
			id          TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL,
			record_id   TEXT NOT NULL,
			genre       TEXT NOT NULL,
			added_at    TIMESTAMPTZ NOT NULL,
			received_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

func (s *PostgresStore) RecordAdded(ctx context.Context, e Entry) (Entry, error) {
	e.ID = uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO activity_entries (id, customer_id, record_id, genre, added_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, customer_id, record_id, genre, added_at, received_at
	`, e.ID, e.CustomerID, e.RecordID, e.Genre, e.AddedAt)

	var out Entry
	err := row.Scan(&out.ID, &out.CustomerID, &out.RecordID, &out.Genre, &out.AddedAt, &out.ReceivedAt)
	return out, err
}

func (s *PostgresStore) RecentFeed(ctx context.Context, limit int) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, customer_id, record_id, genre, added_at, received_at
		FROM activity_entries
		ORDER BY received_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.CustomerID, &e.RecordID, &e.Genre, &e.AddedAt, &e.ReceivedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *PostgresStore) GenreCounts(ctx context.Context) ([]GenreCount, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT genre, COUNT(*) AS count
		FROM activity_entries
		GROUP BY genre
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []GenreCount
	for rows.Next() {
		var gc GenreCount
		if err := rows.Scan(&gc.Genre, &gc.Count); err != nil {
			return nil, err
		}
		counts = append(counts, gc)
	}
	return counts, rows.Err()
}
