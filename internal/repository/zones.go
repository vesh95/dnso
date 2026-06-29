package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type Zone struct {
	Id   uint64
	Name string
	TTL  int64
}

type ZoneRepository interface {
	Get(ctx context.Context, name string) (*Zone, error)
	GetId(ctx context.Context, id uint64) (*Zone, error)
	Add(ctx context.Context, name string, ttl int64) (*Zone, error)
	Update(ctx context.Context, name string, ttl int64) (*Zone, error)
	Delete(ctx context.Context, name string) (bool, error)
}

type ZoneStorage struct {
	db *sql.DB
}

func NewZoneStorage(db *sql.DB) *ZoneStorage {
	return &ZoneStorage{db}
}

func (s *ZoneStorage) Get(ctx context.Context, name string) (*Zone, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, name, ttl FROM zones WHERE name = $1 LIMIT 1", name)

	if err := row.Err(); err != nil {
		return nil, err
	}

	z := &Zone{}
	err := row.Scan(&z.Id, &z.Name, &z.TTL)
	if err != nil {
		return nil, err
	}

	return z, nil
}

func (s *ZoneStorage) GetId(ctx context.Context, id uint64) (*Zone, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, name, ttl FROM zones WHERE id = $1 LIMIT 1", id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	z := &Zone{}
	err := row.Scan(&z.Id, &z.Name, &z.TTL)
	if err != nil {
		return nil, err
	}

	return z, nil
}

func (s *ZoneStorage) Add(ctx context.Context, name string, ttl int64) (*Zone, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO zones (name, ttl) VALUES ($1, $2)", name, ttl)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		return nil, err
	}

	return s.Get(ctx, name)
}

func (s *ZoneStorage) Update(ctx context.Context, name string, ttl int64) (*Zone, error) {
	res, err := s.db.ExecContext(ctx, "UPDATE zones SET ttl = $1 WHERE name = $2", ttl, name)
	if err != nil {
		return nil, err
	}

	affected, err := res.RowsAffected()
	if affected == 0 || err != nil {
		return nil, fmt.Errorf("rows not affected: %w", err)
	}

	return s.Get(ctx, name)
}

func (s *ZoneStorage) Delete(ctx context.Context, name string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM zones WHERE name = $1", name)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, fmt.Errorf("rows not affected")
	}

	return true, nil
}
