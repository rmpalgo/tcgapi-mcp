package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIBaseURL string
	APITimeout time.Duration
	CacheDir   string
	CacheMaxMB int
	LogLevel   string
	PageSize   int
}

type LookupFunc func(string) (string, bool)

func Load() (Config, error) {
	return LoadFromLookup(os.LookupEnv)
}

func LoadFromLookup(lookup LookupFunc) (Config, error) {
	cfg := Config{
		APIBaseURL: lookupString(lookup, "TCG_API_URL", "https://tcgtracking.com/tcgapi/v1"),
		CacheDir:   expandHomeDir(lookupString(lookup, "TCG_CACHE_DIR", "")),
		LogLevel:   lookupString(lookup, "TCG_LOG_LEVEL", "info"),
	}

	timeout, err := lookupDuration(lookup, "TCG_API_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	cfg.APITimeout = timeout

	cacheMaxMB, err := lookupInt(lookup, "TCG_CACHE_MAX_MB", 256)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	cfg.CacheMaxMB = cacheMaxMB

	pageSize, err := lookupInt(lookup, "TCG_PAGE_SIZE", 50)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	cfg.PageSize = pageSize

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.APIBaseURL == "" {
		return fmt.Errorf("TCG_API_URL must not be empty")
	}

	u, err := url.Parse(c.APIBaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("TCG_API_URL must be a valid absolute URL")
	}

	if c.APITimeout <= 0 {
		return fmt.Errorf("TCG_API_TIMEOUT must be > 0")
	}

	if c.CacheMaxMB <= 0 {
		return fmt.Errorf("TCG_CACHE_MAX_MB must be > 0")
	}

	if c.PageSize <= 0 {
		return fmt.Errorf("TCG_PAGE_SIZE must be > 0")
	}

	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "warning", "error":
	default:
		return fmt.Errorf("TCG_LOG_LEVEL must be one of debug, info, warn, error")
	}

	return nil
}

func lookupString(lookup LookupFunc, key, fallback string) string {
	if v, ok := lookup(key); ok && v != "" {
		return v
	}
	return fallback
}

func expandHomeDir(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}

	switch path {
	case "~":
		return home
	case "~/":
		return home
	default:
		if strings.HasPrefix(path, "~/") {
			return filepath.Join(home, path[2:])
		}
		return path
	}
}

func lookupInt(lookup LookupFunc, key string, fallback int) (int, error) {
	v, ok := lookup(key)
	if !ok || v == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return n, nil
}

func lookupDuration(lookup LookupFunc, key string, fallback time.Duration) (time.Duration, error) {
	v, ok := lookup(key)
	if !ok || v == "" {
		return fallback, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}

	return d, nil
}
