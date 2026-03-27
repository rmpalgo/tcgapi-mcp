package tcgapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCategoriesRevalidatesExpiredCache(t *testing.T) {
	t.Parallel()

	requests := 0
	now := time.Now()
	cache := NewMemoryCache(1 << 20)
	client, err := NewClient(Dependencies{
		BaseURL: "https://example.invalid",
		HTTPClient: &http.Client{Transport: &cachingTransport{
			base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Path != "/categories" {
					return newJSONResponse(r, http.StatusNotFound, `{}`), nil
				}

				headers := make(http.Header)
				headers.Set("Content-Type", "application/json")
				headers.Set("ETag", `"cats-v1"`)
				if r.Header.Get("If-None-Match") == `"cats-v1"` {
					return &http.Response{
						StatusCode: http.StatusNotModified,
						Header:     headers,
						Body:       io.NopCloser(strings.NewReader("")),
						Request:    r,
					}, nil
				}

				resp := newJSONResponse(r, http.StatusOK, `{"categories":[{"id":1,"name":"Magic","display_name":"Magic: The Gathering","product_count":10,"set_count":2,"api_url":"/1/sets"}]}`)
				resp.Header.Set("ETag", `"cats-v1"`)
				return resp, nil
			}),
			cache:     cache,
			now:       func() time.Time { return now },
			userAgent: "test",
		}},
		Cache:  cache,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	categories, err := client.Categories(context.Background())
	if err != nil {
		t.Fatalf("Categories() first call error = %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("Categories() len = %d, want 1", len(categories))
	}

	now = now.Add(categoriesTTL + time.Second)

	categories, err = client.Categories(context.Background())
	if err != nil {
		t.Fatalf("Categories() second call error = %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("Categories() len = %d, want 1", len(categories))
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
}

func TestSetPricingFiltersProductID(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Dependencies{
		BaseURL: "https://example.invalid",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/3/sets/99/pricing" {
				return newJSONResponse(r, http.StatusNotFound, `{}`), nil
			}

			return newJSONResponse(r, http.StatusOK, `{"set_id":99,"updated":"2026-03-27T08:04:10-04:00","prices":{"100":{"tcg":{"Normal":{"low":1.1,"market":1.2}}},"200":{"tcg":{"Holofoil":{"low":2.1,"market":2.2}}}}}`), nil
		})},
		Cache:  NewMemoryCache(1 << 20),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	productID := 200
	prices, err := client.SetPricing(context.Background(), 3, 99, &productID)
	if err != nil {
		t.Fatalf("SetPricing() error = %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("len(prices) = %d, want 1", len(prices))
	}
	if prices[0].ProductID != 200 {
		t.Fatalf("ProductID = %d, want 200", prices[0].ProductID)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newJSONResponse(req *http.Request, status int, body string) *http.Response {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
