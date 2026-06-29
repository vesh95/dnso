package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordAdd(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	zone := NewZoneStorage(db)
	z, err := zone.Add(context.Background(), "example.com.", 300)
	require.NoError(t, err)

	s := NewRecordStorage(db)
	rec, err := s.Add(context.Background(), z.Id, "example.com.", "A", "192.168.0.1", 36000)

	require.NoError(t, err)
	assert.Equal(t, z.Id, rec.ZoneId)
	assert.Equal(t, "example.com.", rec.Domain)
	assert.Equal(t, "A", rec.Type)
	assert.Equal(t, int64(36000), rec.TTL)
}

func TestRecordGet(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	zone := NewZoneStorage(db)
	z, err := zone.Add(context.Background(), "example.com.", 300)
	require.NoError(t, err)

	s := NewRecordStorage(db)
	rec, err := s.Add(context.Background(), z.Id, "example.com.", "A", "192.168.0.1", 36000)
	require.NoError(t, err)

	records, err := s.Get(context.Background(), "example.com.", "A")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, rec.Id, records[0].Id)
	assert.Equal(t, "example.com.", records[0].Domain)
	assert.Equal(t, "A", records[0].Type)
}

func TestRecordGetNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewRecordStorage(db)
	records, err := s.Get(context.Background(), "nonexistent.com.", "A")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestRecordGetId(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	zone := NewZoneStorage(db)
	z, err := zone.Add(context.Background(), "example.com.", 300)
	require.NoError(t, err)

	s := NewRecordStorage(db)
	rec, err := s.Add(context.Background(), z.Id, "example.com.", "A", "192.168.0.1", 36000)
	require.NoError(t, err)

	got, err := s.GetId(context.Background(), rec.Id)
	require.NoError(t, err)
	assert.Equal(t, rec.Id, got.Id)
	assert.Equal(t, rec.Domain, got.Domain)
	assert.Equal(t, rec.Type, got.Type)
}

func TestRecordGetIdNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewRecordStorage(db)
	_, err := s.GetId(context.Background(), 999)
	require.Error(t, err)
}

func TestRecordUpdate(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	zone := NewZoneStorage(db)
	z, err := zone.Add(context.Background(), "example.com.", 300)
	require.NoError(t, err)

	s := NewRecordStorage(db)
	rec, err := s.Add(context.Background(), z.Id, "example.com.", "A", "192.168.0.1", 36000)
	require.NoError(t, err)

	updated, err := s.Update(context.Background(), rec.Id, z.Id, "example.com.", "AAAA", "::1", 7200)
	require.NoError(t, err)
	assert.Equal(t, rec.Id, updated.Id)
	assert.Equal(t, "AAAA", updated.Type)
	assert.Equal(t, "::1", updated.Rdata)
	assert.Equal(t, int64(7200), updated.TTL)
}

func TestRecordUpdateNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	zone := NewZoneStorage(db)
	z, err := zone.Add(context.Background(), "example.com.", 300)
	require.NoError(t, err)

	s := NewRecordStorage(db)
	_, err = s.Update(context.Background(), 999, z.Id, "example.com.", "A", "192.168.0.1", 36000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rows not affected")
}

func TestRecordDelete(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	zone := NewZoneStorage(db)
	z, err := zone.Add(context.Background(), "example.com.", 300)
	require.NoError(t, err)

	s := NewRecordStorage(db)
	rec, err := s.Add(context.Background(), z.Id, "example.com.", "A", "192.168.0.1", 36000)
	require.NoError(t, err)

	ok, err := s.Delete(context.Background(), rec.Id)
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = s.GetId(context.Background(), rec.Id)
	require.Error(t, err)
}

func TestRecordDeleteNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewRecordStorage(db)
	_, err := s.Delete(context.Background(), 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rows not affected")
}
