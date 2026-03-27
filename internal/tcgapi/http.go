package tcgapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rmpalgo/tcgapi-mcp/internal/config"
)

const (
	metaTTL       = 5 * time.Minute
	categoriesTTL = 7 * 24 * time.Hour
	setsTTL       = 7 * 24 * time.Hour
	productsTTL   = 7 * 24 * time.Hour
	pricingTTL    = 24 * time.Hour
	skusTTL       = 24 * time.Hour
	searchTTL     = 5 * time.Minute
	defaultUA     = "tcgapi-mcp/0.0.0-dev"
)

type cachingTransport struct {
	base      http.RoundTripper
	cache     Cache
	now       func() time.Time
	userAgent string
}

func NewHTTPClient(cfg config.Config, cache Cache) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 20
	transport.MaxIdleConnsPerHost = 10
	transport.MaxConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second

	return &http.Client{
		Timeout: cfg.APITimeout,
		Transport: &cachingTransport{
			base:      transport,
			cache:     cache,
			now:       time.Now,
			userAgent: defaultUA,
		},
	}
}

func (t *cachingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}

	if t.base == nil {
		t.base = http.DefaultTransport
	}
	if t.now == nil {
		t.now = time.Now
	}

	req = req.Clone(req.Context())
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if req.Header.Get("User-Agent") == "" && t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}

	ttl := ttlForPath(req.URL.Path)
	cacheKey := requestCacheKey(req)
	if ttl > 0 && t.cache != nil {
		if entry, ok := lookupFresh(t.cache, t.now, cacheKey); ok {
			return cachedResponse(req, entry, nil), nil
		}

		if entry, ok := t.cache.Get(cacheKey); ok {
			if entry.ETag != "" {
				req.Header.Set("If-None-Match", entry.ETag)
			}
			if entry.LastModified != "" {
				req.Header.Set("If-Modified-Since", entry.LastModified)
			}
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if ttl <= 0 || t.cache == nil {
		return resp, nil
	}

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("read response body: %w", err)
		}
		_ = resp.Body.Close()

		_ = t.cache.Put(cacheKey, CacheEntry{
			Data:         body,
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
			FetchedAt:    t.now(),
			TTL:          ttl,
		})

		return responseWithBody(resp, req, body), nil
	case http.StatusNotModified:
		entry, ok := t.cache.Get(cacheKey)
		if !ok || len(entry.Data) == 0 {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("received 304 for %s without cached body", req.URL.String())
		}
		_ = resp.Body.Close()

		if etag := resp.Header.Get("ETag"); etag != "" {
			entry.ETag = etag
		}
		if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
			entry.LastModified = lastModified
		}
		entry.FetchedAt = t.now()
		entry.TTL = ttl
		_ = t.cache.Put(cacheKey, entry)

		return cachedResponse(req, entry, resp.Header), nil
	default:
		return resp, nil
	}
}

func lookupFresh(cache Cache, now func() time.Time, key string) (CacheEntry, bool) {
	if cache == nil {
		return CacheEntry{}, false
	}

	entry, ok := cache.Get(key)
	if !ok {
		return CacheEntry{}, false
	}

	if entry.TTL <= 0 || now().Sub(entry.FetchedAt) >= entry.TTL {
		return CacheEntry{}, false
	}

	return entry, true
}

func ttlForPath(path string) time.Duration {
	switch {
	case path == "/meta":
		return metaTTL
	case path == "/categories":
		return categoriesTTL
	case strings.HasSuffix(path, "/search"):
		return searchTTL
	case strings.HasSuffix(path, "/pricing"):
		return pricingTTL
	case strings.HasSuffix(path, "/skus"):
		return skusTTL
	case strings.HasSuffix(path, "/sets"):
		return setsTTL
	case strings.Contains(path, "/sets/"):
		return productsTTL
	default:
		return 0
	}
}

func requestCacheKey(req *http.Request) string {
	return req.Method + " " + req.URL.String()
}

func cachedResponse(req *http.Request, entry CacheEntry, headers http.Header) *http.Response {
	header := make(http.Header)
	if headers != nil {
		header = headers.Clone()
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	if entry.ETag != "" {
		header.Set("ETag", entry.ETag)
	}
	if entry.LastModified != "" {
		header.Set("Last-Modified", entry.LastModified)
	}

	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        http.StatusText(http.StatusOK),
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(entry.Data)),
		ContentLength: int64(len(entry.Data)),
		Request:       req,
	}
}

func responseWithBody(resp *http.Response, req *http.Request, body []byte) *http.Response {
	cloned := *resp
	cloned.Header = resp.Header.Clone()
	cloned.Body = io.NopCloser(bytes.NewReader(body))
	cloned.ContentLength = int64(len(body))
	cloned.Request = req
	return &cloned
}
