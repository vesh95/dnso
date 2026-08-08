package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Zone struct {
	Id      uint64
	Name    string
	TTL     int64
	Refresh int64
	Retry   int64
	Expire  int64
	Serial  uint32
}

type ZoneRepository interface {
	Get(ctx context.Context, name string) (*Zone, error)
	GetId(ctx context.Context, id uint64) (*Zone, error)
	GetAll(ctx context.Context) ([]*Zone, error)
	Add(ctx context.Context, name string, ttl, refresh, retry, expire int64) (*Zone, error)
	Update(ctx context.Context, name string, ttl, refresh, retry, expire int64) (*Zone, error)
	Delete(ctx context.Context, name string) (bool, error)
}

type ZoneStorage struct {
	db *sql.DB
}

func NewZoneStorage(db *sql.DB) *ZoneStorage {
	return &ZoneStorage{db}
}

func (s *ZoneStorage) Get(ctx context.Context, name string) (*Zone, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, name, ttl, refresh, retry, expire, serial FROM zones WHERE name = $1", name)

	z := &Zone{}
	err := row.Scan(&z.Id, &z.Name, &z.TTL, &z.Refresh, &z.Retry, &z.Expire, &z.Serial)
	if err != nil {
		return nil, err
	}

	return z, nil
}

func (s *ZoneStorage) GetAll(ctx context.Context) ([]*Zone, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, ttl, refresh, retry, expire, serial FROM zones")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []*Zone
	for rows.Next() {
		z := &Zone{}
		if err := rows.Scan(&z.Id, &z.Name, &z.TTL, &z.Refresh, &z.Retry, &z.Expire, &z.Serial); err != nil {
			return nil, err
		}
		zones = append(zones, z)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return zones, nil
}

func (s *ZoneStorage) GetId(ctx context.Context, id uint64) (*Zone, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, name, ttl, refresh, retry, expire, serial FROM zones WHERE id = $1", id)

	z := &Zone{}
	err := row.Scan(&z.Id, &z.Name, &z.TTL, &z.Refresh, &z.Retry, &z.Expire, &z.Serial)
	if err != nil {
		return nil, err
	}

	return z, nil
}

func (s *ZoneStorage) Add(ctx context.Context, name string, ttl, refresh, retry, expire int64) (*Zone, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO zones (name, ttl, refresh, retry, expire, serial) VALUES ($1, $2, $3, $4, $5, $6)", name, ttl, refresh, retry, expire, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	if id == 0 {
		return nil, fmt.Errorf("last insert id is 0")
	}

	return s.Get(ctx, name)
}

func (s *ZoneStorage) Update(ctx context.Context, name string, ttl, refresh, retry, expire int64) (*Zone, error) {
	res, err := s.db.ExecContext(ctx, "UPDATE zones SET ttl = $1, refresh = $2, retry = $3, expire = $4, serial = $5 WHERE name = $6", ttl, refresh, retry, expire, time.Now().Unix(), name)
	if err != nil {
		return nil, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("affected rows error: %w", err)
	}

	if affected == 0 {
		return nil, fmt.Errorf("rows not affected")
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
		return false, fmt.Errorf("affected rows error: %w", err)
	}
	if affected == 0 {
		return false, fmt.Errorf("rows not affected")
	}

	return true, nil
}
