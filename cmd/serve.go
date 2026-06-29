/*
Copyright © 2026 Eduard Larionov <vesh95.17@ya.ru>
*/
package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dnso/internal/repository"
	"dnso/internal/server"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start DNS server",
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

func runServer() error {
	dbPath := envOrDefault("DNSO_ DB_PATH", "./dnso.db")
	bindAddr := envOrDefault("DNSO_BIND_ADDR", ":5354")
	upstream := envOrDefault("DNSO_UPSTREAM", "8.8.8.8:53")
	enableCache := envOrDefault("DNSO_CACHE", "true") == "true"

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
	})

	// Регистрируем хендлер для всех доменов
	dns.HandleFunc(".", handler.ServeDNS)

	// Запускаем DNS-сервер
	srv := &dns.Server{
		Addr: bindAddr,
		Net:  "udp",
	}

	// Канал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("DNS server listening on %s (upstream: %s)", bindAddr, upstream)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Ждём сигнал
	sig := <-quit
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown
	if err := srv.Shutdown(); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}
