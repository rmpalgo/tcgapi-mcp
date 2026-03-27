package tcgapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	cacheSnapshotFile    = "cache.json"
	cacheSnapshotVersion = 1
)

type CacheEntry struct {
	Data         []byte
	ETag         string
	LastModified string
	FetchedAt    time.Time
	TTL          time.Duration
}

type PersistentCache interface {
	Cache
	LoadFromDisk(string) (int, error)
	SaveToDisk(string) (int, error)
}

type MemoryCache struct {
	mu       sync.RWMutex
	entries  map[string]CacheEntry
	maxBytes int64
	size     int64
}

type cacheSnapshot struct {
	Version int                   `json:"version"`
	SavedAt time.Time             `json:"saved_at"`
	Entries map[string]CacheEntry `json:"entries"`
}

type persistedEntry struct {
	Key   string
	Entry CacheEntry
}

var _ PersistentCache = (*MemoryCache)(nil)

func NewMemoryCache(maxBytes int64) *MemoryCache {
	return &MemoryCache{
		entries:  make(map[string]CacheEntry),
		maxBytes: maxBytes,
	}
}

func (c *MemoryCache) Get(key string) (CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return CacheEntry{}, false
	}

	return cloneCacheEntry(entry), true
}

func (c *MemoryCache) Put(key string, value CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.putLocked(key, value)
}

func (c *MemoryCache) LoadFromDisk(dir string) (int, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return 0, nil
	}

	data, err := os.ReadFile(filepath.Join(dir, cacheSnapshotFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read cache snapshot: %w", err)
	}

	var snapshot cacheSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return 0, fmt.Errorf("decode cache snapshot: %w", err)
	}
	if snapshot.Version != 0 && snapshot.Version != cacheSnapshotVersion {
		return 0, fmt.Errorf("unsupported cache snapshot version %d", snapshot.Version)
	}

	return c.restore(snapshot.Entries), nil
}

func (c *MemoryCache) SaveToDisk(dir string) (int, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return 0, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, fmt.Errorf("create cache dir: %w", err)
	}

	snapshot := cacheSnapshot{
		Version: cacheSnapshotVersion,
		SavedAt: time.Now().UTC(),
		Entries: c.snapshot(),
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("encode cache snapshot: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "cache-*.tmp")
	if err != nil {
		return 0, fmt.Errorf("create temp cache snapshot: %w", err)
	}

	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("write cache snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("close cache snapshot: %w", err)
	}

	targetPath := filepath.Join(dir, cacheSnapshotFile)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		if removeErr := os.Remove(targetPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("replace cache snapshot: %w", removeErr)
		}
		if err := os.Rename(tmpPath, targetPath); err != nil {
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("move cache snapshot into place: %w", err)
		}
	}

	return len(snapshot.Entries), nil
}

func (c *MemoryCache) snapshot() map[string]CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make(map[string]CacheEntry, len(c.entries))
	for key, entry := range c.entries {
		out[key] = cloneCacheEntry(entry)
	}
	return out
}

func (c *MemoryCache) restore(entries map[string]CacheEntry) int {
	list := make([]persistedEntry, 0, len(entries))
	for key, entry := range entries {
		if key == "" {
			continue
		}
		list = append(list, persistedEntry{
			Key:   key,
			Entry: cloneCacheEntry(entry),
		})
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Entry.FetchedAt.Equal(list[j].Entry.FetchedAt) {
			return list[i].Key < list[j].Key
		}
		return list[i].Entry.FetchedAt.After(list[j].Entry.FetchedAt)
	})

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]CacheEntry, len(list))
	c.size = 0

	loaded := 0
	for _, item := range list {
		entrySize := int64(len(item.Entry.Data))
		if c.maxBytes > 0 && entrySize > c.maxBytes {
			continue
		}
		if c.maxBytes > 0 && c.size+entrySize > c.maxBytes {
			continue
		}

		c.entries[item.Key] = item.Entry
		c.size += entrySize
		loaded++
	}

	return loaded
}

func (c *MemoryCache) putLocked(key string, value CacheEntry) error {
	value = cloneCacheEntry(value)
	newSize := int64(len(value.Data))

	if existing, ok := c.entries[key]; ok {
		c.size -= int64(len(existing.Data))
	}

	if c.maxBytes > 0 && newSize > c.maxBytes {
		return nil
	}

	if c.maxBytes > 0 && c.size+newSize > c.maxBytes {
		c.entries = make(map[string]CacheEntry)
		c.size = 0
	}

	c.entries[key] = value
	c.size += newSize
	return nil
}

func cloneCacheEntry(entry CacheEntry) CacheEntry {
	entry.Data = append([]byte(nil), entry.Data...)
	return entry
}
