package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twobit1268/vinylvault/services/catalog/internal/model"
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
		CREATE TABLE IF NOT EXISTS records (
			id         TEXT PRIMARY KEY,
			title      TEXT NOT NULL,
			artist     TEXT NOT NULL,
			genre      TEXT NOT NULL,
			year       INT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

func (s *PostgresStore) Create(ctx context.Context, r model.Record) (model.Record, error) {
	r.ID = uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO records (id, title, artist, genre, year)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, artist, genre, year, created_at
	`, r.ID, r.Title, r.Artist, r.Genre, r.Year)
	return scanRecord(row)
}

func (s *PostgresStore) Get(ctx context.Context, id string) (model.Record, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, title, artist, genre, year, created_at FROM records WHERE id = $1
	`, id)
	rec, err := scanRecord(row)
	if err != nil {
		return model.Record{}, ErrNotFound
	}
	return rec, nil
}

func (s *PostgresStore) List(ctx context.Context) ([]model.Record, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, artist, genre, year, created_at FROM records ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.Record
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(row rowScanner) (model.Record, error) {
	var r model.Record
	err := row.Scan(&r.ID, &r.Title, &r.Artist, &r.Genre, &r.Year, &r.CreatedAt)
	return r, err
}
