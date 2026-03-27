package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromLookupDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFromLookup(func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("LoadFromLookup() error = %v", err)
	}

	if cfg.APIBaseURL != "https://tcgtracking.com/tcgapi/v1" {
		t.Fatalf("APIBaseURL = %q", cfg.APIBaseURL)
	}
	if cfg.PageSize != 50 {
		t.Fatalf("PageSize = %d", cfg.PageSize)
	}
	if cfg.CacheMaxMB != 256 {
		t.Fatalf("CacheMaxMB = %d", cfg.CacheMaxMB)
	}
}

func TestLoadFromLookupInvalidDuration(t *testing.T) {
	t.Parallel()

	_, err := LoadFromLookup(func(key string) (string, bool) {
		if key == "TCG_API_TIMEOUT" {
			return "not-a-duration", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestLoadFromLookupInvalidPageSize(t *testing.T) {
	t.Parallel()

	_, err := LoadFromLookup(func(key string) (string, bool) {
		if key == "TCG_PAGE_SIZE" {
			return "0", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("expected error for invalid page size")
	}
}

func TestLoadFromLookupExpandsCacheDir(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("home directory not available")
	}

	cfg, err := LoadFromLookup(func(key string) (string, bool) {
		if key == "TCG_CACHE_DIR" {
			return "~/.cache/tcgapi-mcp", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("LoadFromLookup() error = %v", err)
	}

	want := filepath.Join(home, ".cache", "tcgapi-mcp")
	if cfg.CacheDir != want {
		t.Fatalf("CacheDir = %q, want %q", cfg.CacheDir, want)
	}
}
