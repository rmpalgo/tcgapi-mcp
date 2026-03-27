package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/rmpalgo/tcgapi-mcp/internal/buildinfo"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi"
)

type fakePersistentCache struct {
	loadDir string
	saveDir string
	loadErr error
	saveErr error
	loaded  bool
	saved   bool
}

func (f *fakePersistentCache) Get(string) (tcgapi.CacheEntry, bool) {
	return tcgapi.CacheEntry{}, false
}

func (f *fakePersistentCache) Put(string, tcgapi.CacheEntry) error {
	return nil
}

func (f *fakePersistentCache) LoadFromDisk(dir string) (int, error) {
	f.loaded = true
	f.loadDir = dir
	return 2, f.loadErr
}

func (f *fakePersistentCache) SaveToDisk(dir string) (int, error) {
	f.saved = true
	f.saveDir = dir
	return 2, f.saveErr
}

type transientCache struct{}

func (transientCache) Get(string) (tcgapi.CacheEntry, bool) {
	return tcgapi.CacheEntry{}, false
}

func (transientCache) Put(string, tcgapi.CacheEntry) error {
	return nil
}

func TestBuildRuntimeFailsOnInvalidConfig(t *testing.T) {
	t.Setenv("TCG_API_URL", "not-a-valid-url")
	t.Setenv("TCG_LOG_LEVEL", "info")

	_, err := buildRuntime(context.Background(), runOptions{
		Stderr: &bytes.Buffer{},
		Build: buildinfo.Info{
			Version: "test",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "TCG_API_URL") {
		t.Fatalf("error = %q, want TCG_API_URL", got)
	}
}

func TestLoadPersistentCacheUsesPersistentCache(t *testing.T) {
	t.Parallel()

	cache := &fakePersistentCache{}
	cacheDir := t.TempDir()

	loaded, ok, err := loadPersistentCache(cache, cacheDir)
	if err != nil {
		t.Fatalf("loadPersistentCache() error = %v", err)
	}
	if !ok {
		t.Fatal("expected persistent cache to be used")
	}
	if loaded != 2 {
		t.Fatalf("loaded = %d, want 2", loaded)
	}
	if !cache.loaded {
		t.Fatal("expected cache to load from disk")
	}
	if cache.loadDir != cacheDir {
		t.Fatalf("loadDir = %q, want %q", cache.loadDir, cacheDir)
	}
}

func TestLoadPersistentCacheSkipsWithoutPersistentSupport(t *testing.T) {
	t.Parallel()

	loaded, ok, err := loadPersistentCache(transientCache{}, t.TempDir())
	if err != nil {
		t.Fatalf("loadPersistentCache() error = %v", err)
	}
	if ok {
		t.Fatal("expected transient cache to be skipped")
	}
	if loaded != 0 {
		t.Fatalf("loaded = %d, want 0", loaded)
	}
}

func TestSavePersistentCacheUsesPersistentCache(t *testing.T) {
	t.Parallel()

	cache := &fakePersistentCache{}
	cacheDir := t.TempDir()

	saved, ok, err := savePersistentCache(cache, cacheDir)
	if err != nil {
		t.Fatalf("savePersistentCache() error = %v", err)
	}
	if !ok {
		t.Fatal("expected persistent cache to be used")
	}
	if saved != 2 {
		t.Fatalf("saved = %d, want 2", saved)
	}
	if !cache.saved {
		t.Fatal("expected cache to save to disk")
	}
	if cache.saveDir != cacheDir {
		t.Fatalf("saveDir = %q, want %q", cache.saveDir, cacheDir)
	}
}

func TestSavePersistentCacheSkipsEmptyDir(t *testing.T) {
	t.Parallel()

	cache := &fakePersistentCache{}

	saved, ok, err := savePersistentCache(cache, "")
	if err != nil {
		t.Fatalf("savePersistentCache() error = %v", err)
	}
	if ok {
		t.Fatal("expected empty cache dir to skip persistence")
	}
	if saved != 0 {
		t.Fatalf("saved = %d, want 0", saved)
	}
	if cache.saved {
		t.Fatal("did not expect SaveToDisk to be called")
	}
}

func TestRuntimeCloseLogsSaveFailureButDoesNotPanic(t *testing.T) {
	t.Parallel()

	cache := &fakePersistentCache{saveErr: errors.New("cache save failed")}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	rt := &runtime{
		logger:   logger,
		cache:    cache,
		cacheDir: t.TempDir(),
	}

	rt.close()

	if !cache.saved {
		t.Fatal("expected cache save to be attempted")
	}
}
