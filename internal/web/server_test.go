package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dnso/internal/repository"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func upDatabase(t *testing.T) *sql.DB {
	conn, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	err = conn.Ping()
	require.NoError(t, err)

	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	require.NoError(t, err)

	wd, err := os.Getwd()
	require.NoError(t, err)

	migrationsPath := filepath.Join(wd, "..", "..", "migrations")
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "sqlite3", driver)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)

	return conn
}

func setupTest(t *testing.T) (*Server, *sql.DB) {
	db := upDatabase(t)
	s := NewServer(db)
	return s, db
}

func seedZone(t *testing.T, db *sql.DB, name string, ttl int64) *repository.Zone {
	z := repository.NewZoneStorage(db)
	zone, err := z.Add(context.Background(), name, ttl)
	require.NoError(t, err)
	return zone
}

func seedRecord(t *testing.T, db *sql.DB, zoneID uint64, domain, rtype, rdata string, ttl int64) *repository.Record {
	r := repository.NewRecordStorage(db)
	rec, err := r.Add(context.Background(), zoneID, domain, rtype, rdata, ttl)
	require.NoError(t, err)
	return rec
}

func request(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}

// --- Zones ---

func TestWeb_ListZones_Empty(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "GET", "/api/zones", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "null\n", w.Body.String())
}

func TestWeb_ListZones(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)
	seedZone(t, db, "test.org.", 600)

	w := request(t, s, "GET", "/api/zones", "")

	assert.Equal(t, http.StatusOK, w.Code)

	var zones []*repository.Zone
	err := json.Unmarshal(w.Body.Bytes(), &zones)
	require.NoError(t, err)
	require.Len(t, zones, 2)
	assert.Equal(t, "example.com.", zones[0].Name)
	assert.Equal(t, int64(300), zones[0].TTL)
	assert.Equal(t, "test.org.", zones[1].Name)
	assert.Equal(t, int64(600), zones[1].TTL)
}

func TestWeb_GetZone(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)

	w := request(t, s, "GET", "/api/zones/example.com.", "")

	assert.Equal(t, http.StatusOK, w.Code)

	var zone repository.Zone
	err := json.Unmarshal(w.Body.Bytes(), &zone)
	require.NoError(t, err)
	assert.Equal(t, "example.com.", zone.Name)
	assert.Equal(t, int64(300), zone.TTL)
}

func TestWeb_GetZone_NotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "GET", "/api/zones/nonexistent.com.", "")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "zone not found")
}

func TestWeb_CreateZone(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "POST", "/api/zones", `{"name":"example.com.","ttl":3600}`)

	assert.Equal(t, http.StatusCreated, w.Code)

	var zone repository.Zone
	err := json.Unmarshal(w.Body.Bytes(), &zone)
	require.NoError(t, err)
	assert.NotZero(t, zone.Id)
	assert.Equal(t, "example.com.", zone.Name)
	assert.Equal(t, int64(3600), zone.TTL)
}

func TestWeb_CreateZone_AddsDot(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "POST", "/api/zones", `{"name":"example.com","ttl":300}`)

	assert.Equal(t, http.StatusCreated, w.Code)

	var zone repository.Zone
	err := json.Unmarshal(w.Body.Bytes(), &zone)
	require.NoError(t, err)
	assert.Equal(t, "example.com.", zone.Name)
}

func TestWeb_CreateZone_EmptyName(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "POST", "/api/zones", `{"name":"","ttl":300}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWeb_CreateZone_InvalidJSON(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "POST", "/api/zones", `not json`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWeb_CreateZone_Duplicate(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)

	w := request(t, s, "POST", "/api/zones", `{"name":"example.com.","ttl":300}`)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestWeb_UpdateZone(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)

	w := request(t, s, "PUT", "/api/zones/example.com.", `{"ttl":600}`)

	assert.Equal(t, http.StatusOK, w.Code)

	var zone repository.Zone
	err := json.Unmarshal(w.Body.Bytes(), &zone)
	require.NoError(t, err)
	assert.Equal(t, "example.com.", zone.Name)
	assert.Equal(t, int64(600), zone.TTL)
}

func TestWeb_UpdateZone_NotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "PUT", "/api/zones/nonexistent.com.", `{"ttl":600}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWeb_DeleteZone(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)

	w := request(t, s, "DELETE", "/api/zones/example.com.", "")

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Проверяем, что зона удалена
	w2 := request(t, s, "GET", "/api/zones/example.com.", "")
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestWeb_DeleteZone_NotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "DELETE", "/api/zones/nonexistent.com.", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Records ---

func TestWeb_ListRecords_Empty(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)

	w := request(t, s, "GET", "/api/zones/example.com./records", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "null\n", w.Body.String())
}

func TestWeb_ListRecords(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	zone := seedZone(t, db, "example.com.", 300)
	seedRecord(t, db, zone.Id, "test.example.com.", "A", "192.168.1.1", 300)
	seedRecord(t, db, zone.Id, "mail.example.com.", "MX", "10 mail.example.com.", 600)

	w := request(t, s, "GET", "/api/zones/example.com./records", "")

	assert.Equal(t, http.StatusOK, w.Code)

	var records []*repository.Record
	err := json.Unmarshal(w.Body.Bytes(), &records)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "test.example.com.", records[0].Domain)
	assert.Equal(t, "A", records[0].Type)
	assert.Equal(t, "mail.example.com.", records[1].Domain)
	assert.Equal(t, "MX", records[1].Type)
}

func TestWeb_ListRecords_ZoneNotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "GET", "/api/zones/nonexistent.com./records", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWeb_CreateRecord(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)

	w := request(t, s, "POST", "/api/zones/example.com./records",
		`{"domain":"test.example.com.","type":"A","rdata":"192.168.1.1","ttl":300}`)

	assert.Equal(t, http.StatusCreated, w.Code)

	var rec repository.Record
	err := json.Unmarshal(w.Body.Bytes(), &rec)
	require.NoError(t, err)
	assert.NotZero(t, rec.Id)
	assert.Equal(t, "test.example.com.", rec.Domain)
	assert.Equal(t, "A", rec.Type)
	assert.Equal(t, "192.168.1.1", rec.Rdata)
	assert.Equal(t, int64(300), rec.TTL)
}

func TestWeb_CreateRecord_DefaultTTL(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 600)

	w := request(t, s, "POST", "/api/zones/example.com./records",
		`{"domain":"test.example.com.","type":"A","rdata":"192.168.1.1","ttl":0}`)

	assert.Equal(t, http.StatusCreated, w.Code)

	var rec repository.Record
	err := json.Unmarshal(w.Body.Bytes(), &rec)
	require.NoError(t, err)
	// TTL не указан — должен быть взят из зоны
	assert.Equal(t, int64(600), rec.TTL)
}

func TestWeb_CreateRecord_ZoneNotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "POST", "/api/zones/nonexistent.com./records",
		`{"domain":"test.example.com.","type":"A","rdata":"192.168.1.1","ttl":300}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWeb_CreateRecord_MissingFields(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	seedZone(t, db, "example.com.", 300)

	w := request(t, s, "POST", "/api/zones/example.com./records",
		`{"domain":"","type":"A","rdata":"","ttl":300}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWeb_UpdateRecord(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	zone := seedZone(t, db, "example.com.", 300)
	rec := seedRecord(t, db, zone.Id, "test.example.com.", "A", "192.168.1.1", 300)

	body := fmt.Sprintf(
		`{"zone_id":%d,"domain":"test.example.com.","type":"AAAA","rdata":"::1","ttl":600}`,
		zone.Id,
	)
	w := request(t, s, "PUT", fmt.Sprintf("/api/records/%d", rec.Id), body)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated repository.Record
	err := json.Unmarshal(w.Body.Bytes(), &updated)
	require.NoError(t, err)
	assert.Equal(t, rec.Id, updated.Id)
	assert.Equal(t, "AAAA", updated.Type)
	assert.Equal(t, "::1", updated.Rdata)
	assert.Equal(t, int64(600), updated.TTL)
}

func TestWeb_UpdateRecord_NotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	zone := seedZone(t, db, "example.com.", 300)

	body := fmt.Sprintf(
		`{"zone_id":%d,"domain":"test.example.com.","type":"A","rdata":"192.168.1.1","ttl":300}`,
		zone.Id,
	)
	w := request(t, s, "PUT", "/api/records/999", body)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWeb_UpdateRecord_InvalidID(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "PUT", "/api/records/abc",
		`{"zone_id":1,"domain":"test.example.com.","type":"A","rdata":"192.168.1.1","ttl":300}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWeb_DeleteRecord(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	zone := seedZone(t, db, "example.com.", 300)
	rec := seedRecord(t, db, zone.Id, "test.example.com.", "A", "192.168.1.1", 300)

	w := request(t, s, "DELETE", fmt.Sprintf("/api/records/%d", rec.Id), "")

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestWeb_DeleteRecord_NotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "DELETE", "/api/records/999", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWeb_DeleteRecord_InvalidID(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "DELETE", "/api/records/abc", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Static files ---

func TestWeb_IndexPage(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "GET", "/", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "DNSO")
	assert.Contains(t, w.Body.String(), "Зоны")
}

func TestWeb_StaticCSS(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "GET", "/static/style.css", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "body")
	assert.Contains(t, w.Body.String(), "background")
}

func TestWeb_StaticJS(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "GET", "/static/app.js", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "loadZones")
	assert.Contains(t, w.Body.String(), "showRecords")
}

func TestWeb_NotFound(t *testing.T) {
	s, db := setupTest(t)
	defer db.Close()

	w := request(t, s, "GET", "/nonexistent", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}