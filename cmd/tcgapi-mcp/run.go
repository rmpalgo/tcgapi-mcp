package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/rmpalgo/tcgapi-mcp/internal/analysis"
	"github.com/rmpalgo/tcgapi-mcp/internal/buildinfo"
	"github.com/rmpalgo/tcgapi-mcp/internal/catalog"
	"github.com/rmpalgo/tcgapi-mcp/internal/config"
	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/logging"
	"github.com/rmpalgo/tcgapi-mcp/internal/server"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi"
)

type runOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Args   []string
	Build  buildinfo.Info
}

type runtime struct {
	logger   *slog.Logger
	cache    tcgapi.Cache
	cacheDir string
	server   *server.Server
}

func runServer(ctx context.Context, opts runOptions) error {
	rt, err := buildRuntime(ctx, opts)
	if err != nil {
		return err
	}
	defer rt.close()

	return rt.server.ServeStdio(ctx, opts.Stdin, opts.Stdout)
}

func buildRuntime(ctx context.Context, opts runOptions) (_ *runtime, err error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger := logging.New(cfg.LogLevel, opts.Stderr)
	logger.Info("starting tcgapi-mcp", "build", opts.Build.String())

	cache := tcgapi.NewMemoryCache(int64(cfg.CacheMaxMB) << 20)
	rt := &runtime{
		logger:   logger,
		cache:    cache,
		cacheDir: cfg.CacheDir,
	}
	defer func() {
		if err != nil {
			rt.close()
		}
	}()

	if loaded, ok, err := loadPersistentCache(cache, cfg.CacheDir); ok {
		if err != nil {
			logger.Warn("failed to load cache from disk", "dir", cfg.CacheDir, "err", err)
		} else {
			logger.Info("loaded cache from disk", "dir", cfg.CacheDir, "entries", loaded)
		}
	}

	httpClient := tcgapi.NewHTTPClient(cfg, cache)

	apiClient, err := tcgapi.NewClient(tcgapi.Dependencies{
		BaseURL:    cfg.APIBaseURL,
		HTTPClient: httpClient,
		Cache:      cache,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create upstream client: %w", err)
	}

	categories, err := apiClient.Categories(ctx)
	if err != nil {
		return nil, fmt.Errorf("load categories: %w", err)
	}

	if meta, err := apiClient.Meta(ctx); err == nil {
		logger.Info("connected to upstream API", "version", meta.Version, "categories", meta.TotalCategories, "sets", meta.TotalSets)
	} else {
		logger.Warn("failed to load upstream meta", "err", err)
	}

	resolver := catalog.NewResolver(categories, catalog.DefaultAliases())
	analyzer, err := analysis.New(analysis.Dependencies{
		API: apiClient,
		Categories: func(ctx context.Context) ([]domain.Category, error) {
			cats := resolver.Categories()
			if len(cats) > 0 {
				return cats, nil
			}
			return apiClient.Categories(ctx)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build analyzer: %w", err)
	}

	srv, err := server.New(server.Dependencies{
		Logger:   logger,
		API:      apiClient,
		Analyzer: analyzer,
		Resolver: resolver,
		PageSize: cfg.PageSize,
		Build:    opts.Build,
	})
	if err != nil {
		return nil, fmt.Errorf("build server: %w", err)
	}

	rt.server = srv
	return rt, nil
}

func (r *runtime) close() {
	if r == nil {
		return
	}

	saved, ok, err := savePersistentCache(r.cache, r.cacheDir)
	if !ok {
		return
	}
	if err != nil {
		r.logger.Warn("failed to save cache to disk", "dir", r.cacheDir, "err", err)
		return
	}
	r.logger.Info("saved cache to disk", "dir", r.cacheDir, "entries", saved)
}

func loadPersistentCache(cache tcgapi.Cache, dir string) (int, bool, error) {
	persistentCache, ok := persistentCacheFor(cache, dir)
	if !ok {
		return 0, false, nil
	}

	loaded, err := persistentCache.LoadFromDisk(dir)
	return loaded, true, err
}

func savePersistentCache(cache tcgapi.Cache, dir string) (int, bool, error) {
	persistentCache, ok := persistentCacheFor(cache, dir)
	if !ok {
		return 0, false, nil
	}

	saved, err := persistentCache.SaveToDisk(dir)
	return saved, true, err
}

func persistentCacheFor(cache tcgapi.Cache, dir string) (tcgapi.PersistentCache, bool) {
	if strings.TrimSpace(dir) == "" {
		return nil, false
	}
	persistentCache, ok := cache.(tcgapi.PersistentCache)
	if !ok {
		return nil, false
	}
	return persistentCache, true
}
