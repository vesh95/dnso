package config

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"slices"
	"strings"
)

type ServeDNSConfig struct {
	BindAddr      string
	UpstreamAddrs []string
}

type ServeWebConfig struct {
	BindAddr string
}

type ServeConfig struct {
	DatabasePath string
	LogLevel     slog.Level
	Dns          ServeDNSConfig
	Web          ServeWebConfig
	Metrics      ServeWebConfig
}

func ParseEnv() (ServeConfig, error) {
	upstreams, err := parseUpstreams(envOrDefault("DNSO_UPSTREAMS", "8.8.8.8:53"))
	var cfg ServeConfig
	if err != nil {
		return cfg, fmt.Errorf("upstreams load error: %w", err)
	}
	cfg = ServeConfig{
		DatabasePath: envOrDefault("DNSO_DB_PATH", "./dnso.db"),
		LogLevel:     logLevelFromString(envOrDefault("LOG_LEVEL", "info")),
		Dns: ServeDNSConfig{
			BindAddr:      envOrDefault("DNSO_BIND_ADDR", ":53"),
			UpstreamAddrs: upstreams,
		},
		Web: ServeWebConfig{
			BindAddr: envOrDefault("DNSO_WEB_ADDR", ":8080"),
		},
		Metrics: ServeWebConfig{
			BindAddr: envOrDefault("DNSO_METRICS_ADDR", ":3010"),
		},
	}

	return cfg, err
}

// envOrDefault возвращает значение из указанной в `key` переменной окружения,
// если значение не задано - возвращает `default`
func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// logLevelFromString преобразует текстовое представление уровня логирования в slog формат
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

// parseUpstreams принимает строку с адресами upstream-серверов, разделёнными запятыми,
// и возвращает слайс строк с корректными адресами и портами по умолчанию, если не указаны.
func parseUpstreams(adresses string) ([]string, error) {
	result := []string{}
	rawAddrs := strings.Split(adresses, ",")
	if len(rawAddrs) == 0 {
		return []string{}, nil
	}

	for _, v := range rawAddrs {
		v = strings.TrimSpace(v)

		if len(v) == 0 {
			continue
		}

		addr, err := netip.ParseAddrPort(v)
		var addrString string
		if err != nil {
			parseAddr, err := netip.ParseAddr(v)
			if err != nil {
				return []string{}, fmt.Errorf("Invalid upstream address: %s, skipping. Error: %v", v, err)
			} else {
				addrString = netip.AddrPortFrom(parseAddr, 53).String()
			}
		} else {
			addrString = addr.String()
		}

		if slices.Contains(result, addrString) {
			continue
		}

		result = append(result, addrString)
	}

	return result, nil
}
