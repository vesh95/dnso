/*
Copyright © 2026 Eduard Larionov <vesh95.17@ya.ru>
*/
package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"dnso/internal/config"
	"dnso/internal/metrics"
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

func runServer() error {
	config, err := config.ParseEnv()
	if err != nil {
		return fmt.Errorf("Error while read config: %s", err.Error())
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: config.LogLevel}))

	logger.Info("Apply migrations")
	err = runMigrateUp()
	if err != nil {
		return fmt.Errorf("Failed to apply migrations: %w", err)
	}

	metricsRegistry := metrics.NewRegistry()

	db, err := openDB(config.DatabasePath)
	if err != nil {
		return fmt.Errorf("error while database connct: %s", err.Error())
	}
	defer db.Close()

	zoneStorage := repository.NewZoneStorage(db)
	recordStorage := repository.NewRecordStorage(db)
	cache := server.NewDNSCache(metricsRegistry)
	dnsClient := server.NewExchanger(config.Dns.UpstreamAddrs, logger.With("handler_type", "upstream client"))
	handler := server.NewHandler(&server.HandlerConfig{
		Client:        dnsClient,
		ZoneStorage:   zoneStorage,
		RecordStorage: recordStorage,
		Cache:         cache,
		Logger:        logger.With("handler_type", "dns"),
	})

	dns.HandleFunc(".", handler.ServeDNS)

	srv := &dns.Server{
		Addr: config.Dns.BindAddr,
		Net:  "udp",
	}

	webServer := web.NewServer(db, logger.With("handler_type", "web"), handler.RefreshZones, metricsRegistry)
	httpServer := &http.Server{
		Addr:    config.Web.BindAddr,
		Handler: webServer,
	}

	metricsServer := &http.Server{
		Addr:    config.Metrics.BindAddr,
		Handler: metricsRegistry,
	}

	// Канал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Prometheus metrics registry starts", "address", config.Metrics.BindAddr)
		if err := metricsServer.ListenAndServe(); err != nil {
			logger.Error("Error while start prometheus handler", "error", err.Error())
		}
	}()

	// Запускаем DNS-сервер
	go func() {
		log.Printf("DNS server listening on %s (upstream: %s)", config.Dns.BindAddr, strings.Join(config.Dns.UpstreamAddrs, ", "))
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("Failed to start DNS server", "error", err.Error())
		}
	}()

	// Запускаем веб-сервер
	go func() {
		log.Printf("Web interface listening on %s", config.Web.BindAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start web server", "error", err.Error())
		}
	}()

	sig := <-quit
	log.Printf("Received signal %v, shutting down...", sig)

	if err := srv.Shutdown(); err != nil {
		logger.Error("DNS server shutdown", "error", err)
	}
	if err := httpServer.Close(); err != nil {
		logger.Error("web server shutdown", "error", err)
	}

	if err := metricsServer.Close(); err != nil {
		logger.Error("metrics server shutdown", "error", err)
	}

	logger.Info("Shutdown cache background processes...")
	cache.Shutdown()

	logger.Info("Server stopped gracefully")
	return nil
}
