package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZoneAdd(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	z, err := s.Add(context.Background(), "example.com.", 300, 3600, 300, 86400)

	require.NoError(t, err)
	assert.NotZero(t, z.Id)
	assert.Equal(t, "example.com.", z.Name)
	assert.Equal(t, int64(300), z.TTL)
	assert.Equal(t, int64(3600), z.Refresh)
	assert.Equal(t, int64(300), z.Retry)
	assert.Equal(t, int64(86400), z.Expire)
}

func TestZoneAddDuplicate(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	_, err := s.Add(context.Background(), "example.com.", 300, 3600, 300, 86400)
	require.NoError(t, err)

	_, err = s.Add(context.Background(), "example.com.", 300, 3600, 300, 86400)
	require.Error(t, err)

}

func TestZoneGet(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	z, err := s.Add(context.Background(), "example.com.", 300, 3601, 301, 86401)
	require.NoError(t, err)

	got, err := s.Get(context.Background(), "example.com.")
	require.NoError(t, err)
	assert.Equal(t, z.Id, got.Id)
	assert.Equal(t, z.Name, got.Name)
	assert.Equal(t, z.TTL, got.TTL)
}

func TestZoneGetNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	_, err := s.Get(context.Background(), "nonexistent.com.")
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestZoneGetId(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	z, err := s.Add(context.Background(), "example.com.", 300, 3600, 300, 86400)
	require.NoError(t, err)

	got, err := s.GetId(context.Background(), z.Id)
	require.NoError(t, err)
	assert.Equal(t, z.Id, got.Id)
	assert.Equal(t, z.Name, got.Name)
	assert.Equal(t, z.TTL, got.TTL)
}

func TestZoneGetIdNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	_, err := s.GetId(context.Background(), 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestZoneUpdate(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	z, err := s.Add(context.Background(), "example.com.", 300, 3600, 300, 86400)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	updated, err := s.Update(context.Background(), "example.com.", 600, 3601, 301, 86401)
	require.NoError(t, err)
	assert.Equal(t, int64(600), updated.TTL)
	assert.Equal(t, "example.com.", updated.Name)
	assert.Equal(t, int64(3601), updated.Refresh)
	assert.Equal(t, int64(301), updated.Retry)
	assert.Equal(t, int64(86401), updated.Expire)

	assert.Greater(t, updated.Serial, z.Serial)
}

func TestZoneUpdateNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	_, err := s.Update(context.Background(), "nonexistent.com.", 300, 3600, 300, 86400)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rows not affected")
}

func TestZoneDelete(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	_, err := s.Add(context.Background(), "example.com.", 300, 3600, 300, 86400)
	require.NoError(t, err)

	ok, err := s.Delete(context.Background(), "example.com.")
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = s.Get(context.Background(), "example.com.")
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestZoneDeleteNotFound(t *testing.T) {
	db := upDatabase(t)
	defer db.Close()

	s := NewZoneStorage(db)
	_, err := s.Delete(context.Background(), "nonexistent.com.")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rows not affected")
}
