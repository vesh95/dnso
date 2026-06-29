package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type Record struct {
	Id     uint64
	ZoneId uint64
	Domain string
	Type   string
	Rdata  string
	TTL    int64
}

type RecordRepository interface {
	Get(ctx context.Context, domain string) ([]*Record, error)
	GetId(ctx context.Context, id uint64) (*Record, error)
	Add(ctx context.Context, ZoneId uint64, domain, rtype, rdata string, ttl int64) (*Record, error)
	Update(ctx context.Context, id uint64, zoneId uint64, domain, rtype, rdata string, ttl int64) (*Record, error)
	Delete(ctx context.Context, id uint64) (bool, error)
}

type RecordStorage struct {
	db *sql.DB
}

func NewRecordStorage(db *sql.DB) *RecordStorage {
	return &RecordStorage{db}
}

func (s *RecordStorage) Get(ctx context.Context, domain, rtype string) (records []*Record, err error) {
	records = make([]*Record, 0)
	rows, err := s.db.QueryContext(ctx, "SELECT id, zone_id, domain, type, rdata, ttl FROM records WHERE domain = $1 and type = $2", domain, rtype)
	if err != nil {
		return
	}
	defer rows.Close()

	if err = rows.Err(); err != nil {
		return
	}

	for rows.Next() {
		r := &Record{}

		err = rows.Scan(&r.Id, &r.ZoneId, &r.Domain, &r.Type, &r.Rdata, &r.TTL)
		if err != nil {
			return
		}

		records = append(records, r)
	}

	return records, err
}

func (s *RecordStorage) GetId(ctx context.Context, id uint64) (*Record, error) {
	row := s.db.QueryRowContext(ctx, "SELECT id, zone_id, domain, type, rdata, ttl FROM records WHERE id = $1 LIMIT 1", id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	r := &Record{}
	err := row.Scan(&r.Id, &r.ZoneId, &r.Domain, &r.Type, &r.Rdata, &r.TTL)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s *RecordStorage) Add(ctx context.Context, ZoneId uint64, domain, rtype, rdata string, ttl int64) (*Record, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO records (zone_id, domain, type, rdata, ttl) VALUES ($1, $2, $3, $4, $5)",
		ZoneId, domain, rtype, rdata, ttl,
	)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetId(ctx, uint64(id))
}

func (s *RecordStorage) Update(ctx context.Context, id uint64, zoneId uint64, domain, rtype, rdata string, ttl int64) (*Record, error) {
	res, err := s.db.ExecContext(ctx, "UPDATE records SET zone_id = $1, domain = $2, type = $3, rdata = $4, ttl = $5 WHERE id = $6",
		zoneId, domain, rtype, rdata, ttl, id,
	)
	if err != nil {
		return nil, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, fmt.Errorf("rows not affected")
	}

	return s.GetId(ctx, id)
}

func (s *RecordStorage) Delete(ctx context.Context, id uint64) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM records WHERE id = $1", id)
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
