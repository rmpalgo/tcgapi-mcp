package tcgapi

import (
	"bytes"
	"testing"
	"time"
)

func TestMemoryCacheSaveAndLoadFromDisk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := NewMemoryCache(1 << 20)

	firstFetchedAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	secondFetchedAt := firstFetchedAt.Add(10 * time.Minute)

	if err := source.Put("GET https://example.invalid/meta", CacheEntry{
		Data:         []byte(`{"version":"1.0"}`),
		ETag:         `"meta-v1"`,
		LastModified: "Thu, 27 Mar 2026 12:00:00 GMT",
		FetchedAt:    firstFetchedAt,
		TTL:          5 * time.Minute,
	}); err != nil {
		t.Fatalf("Put(meta) error = %v", err)
	}
	if err := source.Put("GET https://example.invalid/categories", CacheEntry{
		Data:         []byte(`{"categories":[{"id":1}]}`),
		ETag:         `"cats-v1"`,
		LastModified: "Thu, 27 Mar 2026 12:10:00 GMT",
		FetchedAt:    secondFetchedAt,
		TTL:          7 * 24 * time.Hour,
	}); err != nil {
		t.Fatalf("Put(categories) error = %v", err)
	}

	saved, err := source.SaveToDisk(dir)
	if err != nil {
		t.Fatalf("SaveToDisk() error = %v", err)
	}
	if saved != 2 {
		t.Fatalf("SaveToDisk() saved = %d, want 2", saved)
	}

	restored := NewMemoryCache(1 << 20)
	loaded, err := restored.LoadFromDisk(dir)
	if err != nil {
		t.Fatalf("LoadFromDisk() error = %v", err)
	}
	if loaded != 2 {
		t.Fatalf("LoadFromDisk() loaded = %d, want 2", loaded)
	}

	metaEntry, ok := restored.Get("GET https://example.invalid/meta")
	if !ok {
		t.Fatal("expected restored meta entry")
	}
	if !bytes.Equal(metaEntry.Data, []byte(`{"version":"1.0"}`)) {
		t.Fatalf("meta data = %q", string(metaEntry.Data))
	}
	if metaEntry.ETag != `"meta-v1"` {
		t.Fatalf("meta etag = %q", metaEntry.ETag)
	}
	if !metaEntry.FetchedAt.Equal(firstFetchedAt) {
		t.Fatalf("meta fetched_at = %v, want %v", metaEntry.FetchedAt, firstFetchedAt)
	}

	categoriesEntry, ok := restored.Get("GET https://example.invalid/categories")
	if !ok {
		t.Fatal("expected restored categories entry")
	}
	if !bytes.Equal(categoriesEntry.Data, []byte(`{"categories":[{"id":1}]}`)) {
		t.Fatalf("categories data = %q", string(categoriesEntry.Data))
	}
	if categoriesEntry.ETag != `"cats-v1"` {
		t.Fatalf("categories etag = %q", categoriesEntry.ETag)
	}
	if !categoriesEntry.FetchedAt.Equal(secondFetchedAt) {
		t.Fatalf("categories fetched_at = %v, want %v", categoriesEntry.FetchedAt, secondFetchedAt)
	}
}

func TestMemoryCacheLoadFromDiskHonorsMaxBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := NewMemoryCache(1 << 20)

	older := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	newer := older.Add(1 * time.Hour)

	if err := source.Put("older", CacheEntry{
		Data:      []byte("aaaa"),
		FetchedAt: older,
		TTL:       time.Hour,
	}); err != nil {
		t.Fatalf("Put(older) error = %v", err)
	}
	if err := source.Put("newer", CacheEntry{
		Data:      []byte("bbbb"),
		FetchedAt: newer,
		TTL:       time.Hour,
	}); err != nil {
		t.Fatalf("Put(newer) error = %v", err)
	}

	if _, err := source.SaveToDisk(dir); err != nil {
		t.Fatalf("SaveToDisk() error = %v", err)
	}

	restored := NewMemoryCache(4)
	loaded, err := restored.LoadFromDisk(dir)
	if err != nil {
		t.Fatalf("LoadFromDisk() error = %v", err)
	}
	if loaded != 1 {
		t.Fatalf("LoadFromDisk() loaded = %d, want 1", loaded)
	}

	if _, ok := restored.Get("newer"); !ok {
		t.Fatal("expected newer entry to be restored")
	}
	if _, ok := restored.Get("older"); ok {
		t.Fatal("did not expect older entry to fit in restored cache")
	}
}
