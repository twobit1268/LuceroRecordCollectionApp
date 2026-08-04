package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twobit1268/vinylvault/services/collection/internal/model"
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
		CREATE TABLE IF NOT EXISTS collection_entries (
			id          TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL,
			record_id   TEXT NOT NULL,
			added_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_collection_entries_customer ON collection_entries(customer_id);
	`)
	return err
}

func (s *PostgresStore) Add(ctx context.Context, e model.Entry) (model.Entry, error) {
	e.ID = uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO collection_entries (id, customer_id, record_id)
		VALUES ($1, $2, $3)
		RETURNING id, customer_id, record_id, added_at
	`, e.ID, e.CustomerID, e.RecordID)
	return scanEntry(row)
}

func (s *PostgresStore) ListByCustomer(ctx context.Context, customerID string) ([]model.Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, customer_id, record_id, added_at
		FROM collection_entries
		WHERE customer_id = $1
		ORDER BY added_at DESC
	`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *PostgresStore) Remove(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM collection_entries WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(row rowScanner) (model.Entry, error) {
	var e model.Entry
	err := row.Scan(&e.ID, &e.CustomerID, &e.RecordID, &e.AddedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.Entry{}, ErrNotFound
		}
		return model.Entry{}, err
	}
	return e, nil
}
