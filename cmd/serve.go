/*
Copyright © 2026 Eduard Larionov <vesh95.17@ya.ru>
*/
package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"dnso/internal/repository"
	"dnso/internal/server"
	"dnso/internal/web"

	"strings"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start DNS server and web interface",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runServer(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func logLevelFromString(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func runServer() error {
	dbPath := envOrDefault("DNSO_DB_PATH", "./dnso.db")
	bindAddr := envOrDefault("DNSO_BIND_ADDR", ":53")
	upstream := envOrDefault("DNSO_UPSTREAM", "8.8.8.8:53")
	enableCache := envOrDefault("DNSO_CACHE", "true") == "true"
	webAddr := envOrDefault("DNSO_WEB_ADDR", ":8080")
	logLevel := logLevelFromString(envOrDefault("LOG_LEVEL", "info"))

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	fmt.Println("Apply migrations")
	err := runMigrateUp()
	if err != nil {
		return fmt.Errorf("Failed to apply migrations: %w", err)
	}

	// Открываем БД
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Создаём репозитории
	zoneStorage := repository.NewZoneStorage(db)
	recordStorage := repository.NewRecordStorage(db)

	// Создаём кэш
	var cache *server.DNSCache
	if enableCache {
		cache = server.NewDNSCache()
	}

	// Создаём DNS-клиент для upstream
	dnsClient := new(dns.Client)

	// Создаём хендлер
	handler := server.NewHandler(&server.HandlerConfig{
		Client:        dnsClient,
		UpstreamAddr:  upstream,
		ZoneStorage:   zoneStorage,
		RecordStorage: recordStorage,
		Cache:         cache,
		Logger:        logger.With("handler_type", "dns"),
	})

	// Регистрируем хендлер для всех доменов
	dns.HandleFunc(".", handler.ServeDNS)

	// Запускаем DNS-сервер
	srv := &dns.Server{
		Addr: bindAddr,
		Net:  "udp",
	}

	// Создаём веб-сервер
	webServer := web.NewServer(db)
	httpServer := &http.Server{
		Addr:    webAddr,
		Handler: webServer,
	}

	// Канал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем DNS-сервер
	go func() {
		log.Printf("DNS server listening on %s (upstream: %s)", bindAddr, upstream)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start DNS server: %v", err)
		}
	}()

	// Запускаем веб-сервер
	go func() {
		log.Printf("Web interface listening on %s", webAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start web server: %v", err)
		}
	}()

	// Ждём сигнал
	sig := <-quit
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown
	if err := srv.Shutdown(); err != nil {
		return fmt.Errorf("DNS server shutdown error: %w", err)
	}
	if err := httpServer.Close(); err != nil {
		return fmt.Errorf("web server shutdown error: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}
