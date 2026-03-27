package tcgapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi/generated"
)

type API interface {
	Meta(context.Context) (domain.Meta, error)
	Categories(context.Context) ([]domain.Category, error)
	CategorySets(context.Context, int) ([]domain.SetSummary, error)
	SearchSets(context.Context, int, string) ([]domain.SetSummary, error)
	SetProducts(context.Context, int, int) ([]domain.Product, error)
	SetPricing(context.Context, int, int, *int) ([]domain.PricingResult, error)
	SetSKUs(context.Context, int, int, *int) ([]domain.SKUResult, error)
}

type Cache interface {
	Get(key string) (CacheEntry, bool)
	Put(key string, value CacheEntry) error
}

type Dependencies struct {
	BaseURL    string
	HTTPClient *http.Client
	Cache      Cache
	Logger     *slog.Logger
}

type Client struct {
	raw    generated.ClientWithResponsesInterface
	cache  Cache
	logger *slog.Logger
	now    func() time.Time
}

var _ API = (*Client)(nil)

func NewClient(d Dependencies) (*Client, error) {
	httpClient := d.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}

	raw, err := generated.NewClientWithResponses(
		d.BaseURL,
		generated.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("create generated tcg client: %w", err)
	}

	return &Client{
		raw:    raw,
		cache:  d.Cache,
		logger: logger,
		now:    time.Now,
	}, nil
}
