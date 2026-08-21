// Package store provides PostgreSQL access via pgx. Hand-written SQL lives
// next to the code that uses it; there is no ORM and no query builder.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the pgx connection pool. All queries hang off it.
type Store struct {
	pool *pgxpool.Pool
}

// New connects a pool and verifies it with a ping.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Ping reports whether the database is reachable.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Family is a row of the families table.
type Family struct {
	ID         int64  `db:"id"`
	Name       string `db:"name"`
	Descriptor string `db:"descriptor"`
}

// Project is a row of the projects table.
type Project struct {
	ID          int64   `db:"id"`
	FamilyID    int64   `db:"family_id"`
	Slug        string  `db:"slug"`
	Name        string  `db:"name"`
	Description string  `db:"description"`
	Version     *string `db:"version"`
	RepoURL     *string `db:"repo_url"`
}

// ListFamilies returns every family ordered by name.
func (s *Store) ListFamilies(ctx context.Context) ([]Family, error) {
	rows, err := s.pool.Query(ctx, `
		select id, name, descriptor
		from families
		order by name`)
	if err != nil {
		return nil, fmt.Errorf("list families: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Family])
}

// ListProjectsByFamily returns a family's projects ordered by name.
func (s *Store) ListProjectsByFamily(ctx context.Context, familyID int64) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		select id, family_id, slug, name, description, version, repo_url
		from projects
		where family_id = $1
		order by name`, familyID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Project])
}
