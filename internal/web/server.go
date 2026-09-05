package web

import (
	"database/sql"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dnso/internal/repository"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

type Server struct {
	zoneStorage   *repository.ZoneStorage
	recordStorage *repository.RecordStorage
	mux           *http.ServeMux
	tmpl          *template.Template
	logger        *slog.Logger
	metrics       WebServerMetrics
	refreshZones  func()
}

func NewServer(db *sql.DB, logger *slog.Logger, refeshZones func(), metrics WebServerMetrics) *Server {
	s := &Server{
		zoneStorage:   repository.NewZoneStorage(db),
		recordStorage: repository.NewRecordStorage(db),
		mux:           http.NewServeMux(),
		logger:        logger,
		metrics:       metrics,
		refreshZones:  refeshZones,
	}

	// Парсим шаблоны из embed.FS
	s.tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

	// API routes
	s.mux.HandleFunc("GET /api/zones", s.durationMiddleware(s.handleListZones))
	s.mux.HandleFunc("GET /api/zones/{name}", s.durationMiddleware(s.handleGetZone))
	s.mux.HandleFunc("POST /api/zones", s.durationMiddleware(s.handleCreateZone))
	s.mux.HandleFunc("PUT /api/zones/{name}", s.durationMiddleware(s.handleUpdateZone))
	s.mux.HandleFunc("DELETE /api/zones/{name}", s.durationMiddleware(s.handleDeleteZone))

	s.mux.HandleFunc("GET /api/zones/{name}/records", s.durationMiddleware(s.handleListRecords))
	s.mux.HandleFunc("POST /api/zones/{name}/records", s.durationMiddleware(s.handleCreateRecord))
	s.mux.HandleFunc("PUT /api/records/{id}", s.durationMiddleware(s.handleUpdateRecord))
	s.mux.HandleFunc("DELETE /api/records/{id}", s.durationMiddleware(s.handleDeleteRecord))

	// Static files
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// SPA — все остальные пути отдаём index.html
	s.mux.HandleFunc("GET /", s.durationMiddleware(s.handleIndex))

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) durationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Вызываем оригинальный хендлер
		next(w, r)

		duration := time.Since(start).Seconds()
		s.metrics.WebAddRequestDuration(r.Method, r.URL.Path, duration) // Замер скорости
		s.metrics.WebIncRequestsTotal() // 1+ количество запросов (можно посчитать через requset_duration и в дальнейшем не потребуется)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := s.tmpl.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		s.logger.Error("failed to render template", "error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// --- Zones ---

func (s *Server) handleListZones(w http.ResponseWriter, r *http.Request) {
	zones, err := s.zoneStorage.GetAll(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, zones)
}

func (s *Server) handleGetZone(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	zone, err := s.zoneStorage.Get(r.Context(), name)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "zone not found"})
		return
	}
	s.writeJSON(w, http.StatusOK, zone)
}

func (s *Server) handleCreateZone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		TTL     int64  `json:"ttl"`
		Refresh int64  `json:"refresh"`
		Retry   int64  `json:"retry"`
		Expire  int64  `json:"expire"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.TTL <= 0 {
		req.TTL = 300
	}
	// Добавляем точку в конце, если её нет
	if !strings.HasSuffix(req.Name, ".") {
		req.Name += "."
	}

	zone, err := s.zoneStorage.Add(r.Context(), req.Name, req.TTL, req.Refresh, req.Retry, req.Expire)
	if err != nil {
		s.writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	s.refreshZones()
	s.writeJSON(w, http.StatusCreated, zone)
}

func (s *Server) handleUpdateZone(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	var req struct {
		TTL     int64 `json:"ttl"`
		Refresh int64 `json:"refresh"`
		Retry   int64 `json:"retry"`
		Expire  int64 `json:"expire"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.TTL <= 0 {
		req.TTL = 300
	}

	zone, err := s.zoneStorage.Update(r.Context(), name, req.TTL, req.Refresh, req.Retry, req.Expire)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, zone)
}

func (s *Server) handleDeleteZone(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	_, err := s.zoneStorage.Delete(r.Context(), name)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.refreshZones()
	w.WriteHeader(http.StatusNoContent)
}

// --- Records ---

func (s *Server) handleListRecords(w http.ResponseWriter, r *http.Request) {
	zoneName := r.PathValue("name")
	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	zone, err := s.zoneStorage.Get(r.Context(), zoneName)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "zone not found"})
		return
	}

	rows, err := s.recordStorage.GetAllByZoneID(r.Context(), zone.Id)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleCreateRecord(w http.ResponseWriter, r *http.Request) {
	zoneName := r.PathValue("name")
	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	zone, err := s.zoneStorage.Get(r.Context(), zoneName)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "zone not found"})
		return
	}

	var req struct {
		Domain string `json:"domain"`
		Type   string `json:"type"`
		Rdata  string `json:"rdata"`
		TTL    int64  `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	req.Domain = strings.TrimSpace(req.Domain)
	req.Type = strings.ToUpper(strings.TrimSpace(req.Type))
	req.Rdata = strings.TrimSpace(req.Rdata)

	if req.Domain == "" || req.Type == "" || req.Rdata == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain, type, and rdata are required"})
		return
	}
	if req.TTL <= 0 {
		req.TTL = zone.TTL
	}

	record, err := s.recordStorage.Add(r.Context(), zone.Id, req.Domain, req.Type, req.Rdata, req.TTL)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, record)
}

func (s *Server) handleUpdateRecord(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid record id"})
		return
	}

	var req struct {
		ZoneId uint64 `json:"zone_id"`
		Domain string `json:"domain"`
		Type   string `json:"type"`
		Rdata  string `json:"rdata"`
		TTL    int64  `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	req.Domain = strings.TrimSpace(req.Domain)
	req.Type = strings.ToUpper(strings.TrimSpace(req.Type))
	req.Rdata = strings.TrimSpace(req.Rdata)

	if req.Domain == "" || req.Type == "" || req.Rdata == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain, type, and rdata are required"})
		return
	}
	if req.TTL <= 0 {
		req.TTL = 300
	}

	record, err := s.recordStorage.Update(r.Context(), id, req.ZoneId, req.Domain, req.Type, req.Rdata, req.TTL)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleDeleteRecord(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid record id"})
		return
	}

	_, err = s.recordStorage.Delete(r.Context(), id)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("json encode error", "error", err.Error())
	}
}
