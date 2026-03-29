package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rmpalgo/tcgapi-mcp/internal/analysis"
	"github.com/rmpalgo/tcgapi-mcp/internal/buildinfo"
	"github.com/rmpalgo/tcgapi-mcp/internal/catalog"
	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

type setKey struct {
	categoryID int
	setID      int
}

type searchCall struct {
	categoryID int
	query      string
}

type productFilterCall struct {
	categoryID int
	setID      int
	productID  *int
}

type fakeAPI struct {
	meta         domain.Meta
	categories   []domain.Category
	categorySets map[int][]domain.SetSummary
	searchSets   map[int][]domain.SetSummary
	products     map[setKey][]domain.Product
	pricing      map[setKey]domain.PricingSnapshot
	skus         map[setKey]domain.SKUSnapshot

	searchCalls  []searchCall
	pricingCalls []productFilterCall
	skuCalls     []productFilterCall
}

func (f *fakeAPI) Meta(context.Context) (domain.Meta, error) {
	return f.meta, nil
}

func (f *fakeAPI) Categories(context.Context) ([]domain.Category, error) {
	return append([]domain.Category(nil), f.categories...), nil
}

func (f *fakeAPI) CategorySets(_ context.Context, categoryID int) ([]domain.SetSummary, error) {
	return append([]domain.SetSummary(nil), f.categorySets[categoryID]...), nil
}

func (f *fakeAPI) SearchSets(_ context.Context, categoryID int, query string) ([]domain.SetSummary, error) {
	f.searchCalls = append(f.searchCalls, searchCall{
		categoryID: categoryID,
		query:      query,
	})
	return append([]domain.SetSummary(nil), f.searchSets[categoryID]...), nil
}

func (f *fakeAPI) SetProducts(_ context.Context, categoryID, setID int) ([]domain.Product, error) {
	return append([]domain.Product(nil), f.products[setKey{categoryID: categoryID, setID: setID}]...), nil
}

func (f *fakeAPI) SetPricing(_ context.Context, categoryID, setID int, productID *int) (domain.PricingSnapshot, error) {
	f.pricingCalls = append(f.pricingCalls, productFilterCall{
		categoryID: categoryID,
		setID:      setID,
		productID:  cloneInt(productID),
	})

	snapshot := f.pricing[setKey{categoryID: categoryID, setID: setID}]
	products := append([]domain.PricingResult(nil), snapshot.Prices...)
	if productID == nil {
		snapshot.Prices = products
		return snapshot, nil
	}

	filtered := make([]domain.PricingResult, 0, len(products))
	for _, result := range products {
		if result.ProductID == *productID {
			filtered = append(filtered, result)
		}
	}
	snapshot.Prices = filtered
	return snapshot, nil
}

func (f *fakeAPI) SetSKUs(_ context.Context, categoryID, setID int, productID *int) (domain.SKUSnapshot, error) {
	f.skuCalls = append(f.skuCalls, productFilterCall{
		categoryID: categoryID,
		setID:      setID,
		productID:  cloneInt(productID),
	})

	snapshot := f.skus[setKey{categoryID: categoryID, setID: setID}]
	products := append([]domain.SKUResult(nil), snapshot.Products...)
	if productID == nil {
		snapshot.Products = products
		return snapshot, nil
	}

	filtered := make([]domain.SKUResult, 0, len(products))
	for _, result := range products {
		if result.ProductID == *productID {
			filtered = append(filtered, result)
		}
	}
	snapshot.Products = filtered
	return snapshot, nil
}

func TestNewRegistersSurface(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)
	ctx := context.Background()

	tools := collectTools(t, ctx, clientSession)
	if got := len(tools); got != 7 {
		t.Fatalf("len(tools) = %d, want 7", got)
	}

	resources := collectResources(t, ctx, clientSession)
	if got := len(resources); got != 3 {
		t.Fatalf("len(resources) = %d, want 3", got)
	}

	templates := collectResourceTemplates(t, ctx, clientSession)
	if got := len(templates); got != 6 {
		t.Fatalf("len(resource templates) = %d, want 6", got)
	}

	prompts := collectPrompts(t, ctx, clientSession)
	if got := len(prompts); got != 6 {
		t.Fatalf("len(prompts) = %d, want 6", got)
	}
}

func TestInitializeReportsBuildVersion(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)

	result := clientSession.InitializeResult()
	if result == nil {
		t.Fatal("InitializeResult() = nil")
	}
	if result.ServerInfo == nil {
		t.Fatal("InitializeResult().ServerInfo = nil")
	}
	if result.ServerInfo.Name != "tcgapi-mcp" {
		t.Fatalf("ServerInfo.Name = %q, want tcgapi-mcp", result.ServerInfo.Name)
	}
	if result.ServerInfo.Version != "v1.2.3-test" {
		t.Fatalf("ServerInfo.Version = %q, want v1.2.3-test", result.ServerInfo.Version)
	}
}

func TestListCategoriesTool(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "list_categories",
	})
	if err != nil {
		t.Fatalf("CallTool(list_categories) error = %v", err)
	}

	output := decodeStructured[listCategoriesOutput](t, result.StructuredContent)
	if got := len(output.Categories); got != 2 {
		t.Fatalf("len(output.Categories) = %d, want 2", got)
	}
	if output.Categories[0].ID != 1 {
		t.Fatalf("first category ID = %d, want 1", output.Categories[0].ID)
	}
}

func TestSearchSetsResolvesAliasesAndNumericIDs(t *testing.T) {
	t.Parallel()

	api := newFakeAPI()
	_, clientSession, _ := newTestServerWithAPI(t, api)

	for _, category := range []string{"mtg", "1"} {
		result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "search_sets",
			Arguments: map[string]any{
				"category": category,
				"query":    "Alpha",
			},
		})
		if err != nil {
			t.Fatalf("CallTool(search_sets, %q) error = %v", category, err)
		}

		output := decodeStructured[searchSetsOutput](t, result.StructuredContent)
		if got := len(output.Sets); got != 2 {
			t.Fatalf("len(output.Sets) = %d, want 2", got)
		}
	}

	if got := len(api.searchCalls); got != 2 {
		t.Fatalf("len(api.searchCalls) = %d, want 2", got)
	}
	for _, call := range api.searchCalls {
		if call.categoryID != 1 {
			t.Fatalf("search call categoryID = %d, want 1", call.categoryID)
		}
		if call.query != "Alpha" {
			t.Fatalf("search call query = %q, want Alpha", call.query)
		}
	}
}

func TestGetSetProductsPagination(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_set_products",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_set_products default page) error = %v", err)
	}

	pageOne := decodeStructured[getSetProductsOutput](t, result.StructuredContent)
	if pageOne.Pagination.Total != 3 || pageOne.Pagination.Returned != 2 || !pageOne.Pagination.HasMore {
		t.Fatalf("default pagination = %+v, want total=3 returned=2 has_more=true", pageOne.Pagination)
	}

	result, err = clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_set_products",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
			"limit":    2,
			"offset":   2,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_set_products offset page) error = %v", err)
	}

	pageTwo := decodeStructured[getSetProductsOutput](t, result.StructuredContent)
	if pageTwo.Pagination.Returned != 1 || pageTwo.Pagination.Offset != 2 || pageTwo.Pagination.HasMore {
		t.Fatalf("offset pagination = %+v, want returned=1 offset=2 has_more=false", pageTwo.Pagination)
	}
	if got := pageTwo.Items[0].Name; got != "Mox Sapphire" {
		t.Fatalf("pageTwo.Items[0].Name = %q, want Mox Sapphire", got)
	}
}

func TestPricingAndSKUsHonorProductID(t *testing.T) {
	t.Parallel()

	api := newFakeAPI()
	_, clientSession, _ := newTestServerWithAPI(t, api)

	productID := 20
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_set_pricing",
		Arguments: map[string]any{
			"category":   "mtg",
			"set_id":     100,
			"product_id": productID,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_set_pricing) error = %v", err)
	}

	pricing := decodeStructured[getSetPricingOutput](t, result.StructuredContent)
	if got := len(pricing.Prices); got != 1 {
		t.Fatalf("len(pricing.Prices) = %d, want 1", got)
	}
	if pricing.UpdatedAt != "2026-03-27T12:00:00Z" {
		t.Fatalf("pricing.UpdatedAt = %q, want 2026-03-27T12:00:00Z", pricing.UpdatedAt)
	}
	if pricing.Prices[0].ProductID != productID {
		t.Fatalf("pricing.Prices[0].ProductID = %d, want %d", pricing.Prices[0].ProductID, productID)
	}
	if pricing.Prices[0].Product == nil {
		t.Fatal("pricing.Prices[0].Product = nil, want joined product summary")
	}
	if pricing.Prices[0].Product.Name != "Time Walk" || pricing.Prices[0].Product.Number != "1" || pricing.Prices[0].Product.Rarity != "Rare" {
		t.Fatalf("pricing.Prices[0].Product = %+v, want Time Walk #1 Rare", pricing.Prices[0].Product)
	}
	if len(api.pricingCalls) != 1 || api.pricingCalls[0].productID == nil || *api.pricingCalls[0].productID != productID {
		t.Fatalf("pricing calls = %+v, want product_id=%d", api.pricingCalls, productID)
	}

	result, err = clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_set_skus",
		Arguments: map[string]any{
			"category":   "1",
			"set_id":     100,
			"product_id": productID,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_set_skus) error = %v", err)
	}

	skus := decodeStructured[getSetSKUsOutput](t, result.StructuredContent)
	if got := len(skus.Products); got != 1 {
		t.Fatalf("len(skus.Products) = %d, want 1", got)
	}
	if skus.UpdatedAt != "2026-03-27T13:00:00Z" {
		t.Fatalf("skus.UpdatedAt = %q, want 2026-03-27T13:00:00Z", skus.UpdatedAt)
	}
	if skus.Products[0].ProductID != productID {
		t.Fatalf("skus.Products[0].ProductID = %d, want %d", skus.Products[0].ProductID, productID)
	}
	if skus.Products[0].Product == nil {
		t.Fatal("skus.Products[0].Product = nil, want joined product summary")
	}
	if skus.Products[0].Product.Name != "Time Walk" || skus.Products[0].Product.Number != "1" || skus.Products[0].Product.Rarity != "Rare" {
		t.Fatalf("skus.Products[0].Product = %+v, want Time Walk #1 Rare", skus.Products[0].Product)
	}
	if len(api.skuCalls) != 1 || api.skuCalls[0].productID == nil || *api.skuCalls[0].productID != productID {
		t.Fatalf("sku calls = %+v, want product_id=%d", api.skuCalls, productID)
	}
}

func TestPricingAndSKUsJoinProductMetadataByProductID(t *testing.T) {
	t.Parallel()

	api := newFakeAPI()
	set := setKey{categoryID: 1, setID: 100}
	api.products[set] = []domain.Product{
		{ID: 20, SetID: 100, Name: "Time Walk", SetName: "Alpha", Number: "1", Rarity: "Rare"},
		{ID: 100, SetID: 100, Name: "Ancestral Recall", SetName: "Alpha", Number: "54", Rarity: "Rare"},
	}
	api.pricing[set] = domain.PricingSnapshot{
		UpdatedAt: "2026-03-27T12:00:00Z",
		Prices: []domain.PricingResult{
			{ProductID: 100, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(9000)}}, Manapool: map[string]float64{}},
			{ProductID: 20, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(5500)}}, Manapool: map[string]float64{}},
		},
	}
	api.skus[set] = domain.SKUSnapshot{
		UpdatedAt: "2026-03-27T13:00:00Z",
		Products: []domain.SKUResult{
			{ProductID: 100, SKUs: []domain.SKU{{ID: 1001, Condition: "NM", Variant: "N", Language: "en", Market: floatPtr(9000)}}},
			{ProductID: 20, SKUs: []domain.SKU{{ID: 2001, Condition: "NM", Variant: "N", Language: "en", Market: floatPtr(5500)}}},
		},
	}

	_, clientSession, _ := newTestServerWithAPI(t, api)

	pricingResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_set_pricing",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_set_pricing) error = %v", err)
	}

	pricing := decodeStructured[getSetPricingOutput](t, pricingResult.StructuredContent)
	if got := pricing.Prices[0].Product; got == nil || got.ID != 100 || got.Name != "Ancestral Recall" {
		t.Fatalf("pricing.Prices[0].Product = %+v, want product 100 Ancestral Recall", got)
	}
	if got := pricing.Prices[1].Product; got == nil || got.ID != 20 || got.Name != "Time Walk" {
		t.Fatalf("pricing.Prices[1].Product = %+v, want product 20 Time Walk", got)
	}

	skuResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_set_skus",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(get_set_skus) error = %v", err)
	}

	skus := decodeStructured[getSetSKUsOutput](t, skuResult.StructuredContent)
	if got := skus.Products[0].Product; got == nil || got.ID != 100 || got.Name != "Ancestral Recall" {
		t.Fatalf("skus.Products[0].Product = %+v, want product 100 Ancestral Recall", got)
	}
	if got := skus.Products[1].Product; got == nil || got.ID != 20 || got.Name != "Time Walk" {
		t.Fatalf("skus.Products[1].Product = %+v, want product 20 Time Walk", got)
	}
}

func TestAnalyticsToolsReturnDerivedSummaries(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)

	countsResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "summarize_release_counts",
		Arguments: map[string]any{
			"category":             "mtg",
			"year_from":            2000,
			"include_supplemental": false,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(summarize_release_counts) error = %v", err)
	}

	counts := decodeStructured[domain.ReleaseCountsSummary](t, countsResult.StructuredContent)
	if counts.CategoryScope != "Magic: The Gathering" {
		t.Fatalf("CategoryScope = %q, want Magic: The Gathering", counts.CategoryScope)
	}
	if counts.TotalSets != 1 {
		t.Fatalf("TotalSets = %d, want 1", counts.TotalSets)
	}

	insightsResult, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "analyze_set_insights",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
			"top_n":    2,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(analyze_set_insights) error = %v", err)
	}

	insights := decodeStructured[domain.SetInsights](t, insightsResult.StructuredContent)
	if insights.Set.Name != "Alpha" {
		t.Fatalf("Set.Name = %q, want Alpha", insights.Set.Name)
	}
	if got := len(insights.HeuristicNotes); got != 3 {
		t.Fatalf("len(HeuristicNotes) = %d, want 3", got)
	}
	if insights.PricingUpdatedAt != "2026-03-27T12:00:00Z" {
		t.Fatalf("PricingUpdatedAt = %q, want 2026-03-27T12:00:00Z", insights.PricingUpdatedAt)
	}
	if len(insights.TopMarketCards) != 2 {
		t.Fatalf("len(TopMarketCards) = %d, want 2", len(insights.TopMarketCards))
	}
	if insights.ProductKindFilterApplied != domain.ProductKindFilterAll {
		t.Fatalf("ProductKindFilterApplied = %q, want %q", insights.ProductKindFilterApplied, domain.ProductKindFilterAll)
	}
}

func TestAnalyzeSetInsightsToolSupportsSinglesFiltering(t *testing.T) {
	t.Parallel()

	api := newFakeAPI()
	set := setKey{categoryID: 1, setID: 100}
	api.products[set] = []domain.Product{
		{ID: 10, SetID: 100, Name: "Black Lotus", CleanName: "Black Lotus", Number: "233", Rarity: "Rare"},
		{ID: 20, SetID: 100, Name: "Time Walk", CleanName: "Time Walk", Number: "1", Rarity: "Rare"},
		{ID: 30, SetID: 100, Name: "Alpha Booster Box", CleanName: "Alpha Booster Box"},
		{ID: 40, SetID: 100, Name: "Code Card - Alpha Booster Pack", CleanName: "Code Card Alpha Booster Pack", Rarity: "Code Card"},
	}
	api.pricing[set] = domain.PricingSnapshot{
		UpdatedAt: "2026-03-27T12:00:00Z",
		Prices: []domain.PricingResult{
			{ProductID: 10, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(11000)}}, Manapool: map[string]float64{}},
			{ProductID: 20, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(5500)}}, Manapool: map[string]float64{}},
			{ProductID: 30, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(15000)}}, Manapool: map[string]float64{}},
			{ProductID: 40, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(200)}}, Manapool: map[string]float64{}},
		},
	}

	_, clientSession, _ := newTestServerWithAPI(t, api)

	minMarketPrice := 1000.0
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "analyze_set_insights",
		Arguments: map[string]any{
			"category":            "1",
			"set_id":              100,
			"top_n":               10,
			"product_kind_filter": "single_like",
			"min_market_price":    minMarketPrice,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(analyze_set_insights filtered) error = %v", err)
	}

	insights := decodeStructured[domain.SetInsights](t, result.StructuredContent)
	if insights.ProductKindFilterApplied != domain.ProductKindFilterSingleLike {
		t.Fatalf("ProductKindFilterApplied = %q, want %q", insights.ProductKindFilterApplied, domain.ProductKindFilterSingleLike)
	}
	if insights.MinMarketPriceApplied == nil || *insights.MinMarketPriceApplied != minMarketPrice {
		t.Fatalf("MinMarketPriceApplied = %+v, want %f", insights.MinMarketPriceApplied, minMarketPrice)
	}
	if got := len(insights.TopMarketCards); got != 2 {
		t.Fatalf("len(TopMarketCards) = %d, want 2", got)
	}
	if insights.TopMarketCards[0].ProductID != 10 || insights.TopMarketCards[1].ProductID != 20 {
		t.Fatalf("TopMarketCards = %+v, want Black Lotus then Time Walk", insights.TopMarketCards)
	}
	for _, card := range insights.TopMarketCards {
		if card.ProductKind != domain.ProductKindSingleLike {
			t.Fatalf("TopMarketCard.ProductKind = %q, want %q", card.ProductKind, domain.ProductKindSingleLike)
		}
	}
}

func TestAnalyzeSetInsightsToolSupportsFieldSelection(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "analyze_set_insights",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
			"top_n":    2,
			"fields":   []string{"top_market_cards", "top_market_cards"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(analyze_set_insights selective) error = %v", err)
	}

	insights := decodeStructured[map[string]any](t, result.StructuredContent)
	assertContainsKeys(t, insights,
		"set",
		"product_count_total",
		"numbered_card_like_count",
		"pricing_updated_at",
		"sku_updated_at",
		"top_market_cards",
		"product_kind_filter_applied",
	)
	assertOmitsKeys(t, insights,
		"numbering_summary",
		"rarity_breakdown",
		"highest_value_rarity",
		"market_sum_estimate",
		"heuristic_notes",
		"min_market_price_applied",
	)

	topCards, ok := insights["top_market_cards"].([]any)
	if !ok {
		t.Fatalf("top_market_cards type = %T, want []any", insights["top_market_cards"])
	}
	if got := len(topCards); got != 2 {
		t.Fatalf("len(top_market_cards) = %d, want 2", got)
	}
}

func TestAnalyzeSetInsightsToolSupportsMetadataOnlyFieldSelection(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "analyze_set_insights",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
			"fields":   []string{},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(analyze_set_insights metadata-only) error = %v", err)
	}

	insights := decodeStructured[map[string]any](t, result.StructuredContent)
	assertContainsKeys(t, insights,
		"set",
		"product_count_total",
		"numbered_card_like_count",
		"pricing_updated_at",
		"sku_updated_at",
		"product_kind_filter_applied",
	)
	assertOmitsKeys(t, insights,
		"numbering_summary",
		"rarity_breakdown",
		"top_market_cards",
		"highest_value_rarity",
		"market_sum_estimate",
		"heuristic_notes",
		"min_market_price_applied",
	)
}

func TestAnalyzeSetInsightsToolRejectsUnsupportedFields(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "analyze_set_insights",
		Arguments: map[string]any{
			"category": "1",
			"set_id":   100,
			"fields":   []string{"top_market_cards", "unknown_field"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(analyze_set_insights invalid fields) transport error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = %v, want true", result.IsError)
	}
	if got := len(result.Content); got != 1 {
		t.Fatalf("len(result.Content) = %d, want 1", got)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("result.Content[0] type = %T, want *mcp.TextContent", result.Content[0])
	}
	if !strings.Contains(text.Text, "unsupported values [unknown_field]") {
		t.Fatalf("content = %q, want unsupported values detail", text.Text)
	}
	if !strings.Contains(text.Text, "supported values are [numbering_summary, rarity_breakdown, top_market_cards, highest_value_rarity, market_sum_estimate, heuristic_notes]") {
		t.Fatalf("content = %q, want supported fields list", text.Text)
	}
}

func TestAnalyzeSetInsightsToolAppliesFiltersWithSelectedFields(t *testing.T) {
	t.Parallel()

	api := newFakeAPI()
	set := setKey{categoryID: 1, setID: 100}
	api.products[set] = []domain.Product{
		{ID: 10, SetID: 100, Name: "Black Lotus", CleanName: "Black Lotus", Number: "233", Rarity: "Rare"},
		{ID: 20, SetID: 100, Name: "Time Walk", CleanName: "Time Walk", Number: "1", Rarity: "Rare"},
		{ID: 30, SetID: 100, Name: "Alpha Booster Box", CleanName: "Alpha Booster Box"},
		{ID: 40, SetID: 100, Name: "Code Card - Alpha Booster Pack", CleanName: "Code Card Alpha Booster Pack", Rarity: "Code Card"},
	}
	api.pricing[set] = domain.PricingSnapshot{
		UpdatedAt: "2026-03-27T12:00:00Z",
		Prices: []domain.PricingResult{
			{ProductID: 10, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(11000)}}, Manapool: map[string]float64{}},
			{ProductID: 20, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(5500)}}, Manapool: map[string]float64{}},
			{ProductID: 30, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(15000)}}, Manapool: map[string]float64{}},
			{ProductID: 40, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(200)}}, Manapool: map[string]float64{}},
		},
	}

	_, clientSession, _ := newTestServerWithAPI(t, api)

	minMarketPrice := 1000.0
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "analyze_set_insights",
		Arguments: map[string]any{
			"category":            "1",
			"set_id":              100,
			"top_n":               10,
			"product_kind_filter": "single_like",
			"min_market_price":    minMarketPrice,
			"fields":              []string{"top_market_cards", "market_sum_estimate"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool(analyze_set_insights filtered selective) error = %v", err)
	}

	insights := decodeStructured[map[string]any](t, result.StructuredContent)
	assertContainsKeys(t, insights,
		"top_market_cards",
		"market_sum_estimate",
		"product_kind_filter_applied",
		"min_market_price_applied",
	)
	assertOmitsKeys(t, insights,
		"numbering_summary",
		"rarity_breakdown",
		"highest_value_rarity",
		"heuristic_notes",
	)

	if got := insights["product_kind_filter_applied"]; got != string(domain.ProductKindFilterSingleLike) {
		t.Fatalf("product_kind_filter_applied = %v, want %q", got, domain.ProductKindFilterSingleLike)
	}
	if got := insights["min_market_price_applied"]; got != minMarketPrice {
		t.Fatalf("min_market_price_applied = %v, want %f", got, minMarketPrice)
	}

	topCards, ok := insights["top_market_cards"].([]any)
	if !ok {
		t.Fatalf("top_market_cards type = %T, want []any", insights["top_market_cards"])
	}
	if got := len(topCards); got != 2 {
		t.Fatalf("len(top_market_cards) = %d, want 2", got)
	}
	firstCard, ok := topCards[0].(map[string]any)
	if !ok {
		t.Fatalf("top_market_cards[0] type = %T, want map[string]any", topCards[0])
	}
	if got := firstCard["product_id"]; got != float64(10) {
		t.Fatalf("top_market_cards[0].product_id = %v, want 10", got)
	}
}

func TestResourcesReadJSON(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)
	ctx := context.Background()

	meta, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///meta"})
	if err != nil {
		t.Fatalf("ReadResource(meta) error = %v", err)
	}
	metaValue := decodeResource[domain.Meta](t, meta)
	if metaValue.Version != "1.2.3" {
		t.Fatalf("meta version = %q, want 1.2.3", metaValue.Version)
	}

	categories, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///categories"})
	if err != nil {
		t.Fatalf("ReadResource(categories) error = %v", err)
	}
	categoriesValue := decodeResource[listCategoriesOutput](t, categories)
	if got := len(categoriesValue.Categories); got != 2 {
		t.Fatalf("len(categoriesValue.Categories) = %d, want 2", got)
	}

	globalCounts, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///analytics/releases-by-year"})
	if err != nil {
		t.Fatalf("ReadResource(global release counts) error = %v", err)
	}
	globalCountsValue := decodeResource[domain.ReleaseCountsSummary](t, globalCounts)
	if globalCountsValue.TotalSets != 3 {
		t.Fatalf("global release count total = %d, want 3", globalCountsValue.TotalSets)
	}

	categoryCounts, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///1/analytics/releases-by-year"})
	if err != nil {
		t.Fatalf("ReadResource(category release counts) error = %v", err)
	}
	categoryCountsValue := decodeResource[domain.ReleaseCountsSummary](t, categoryCounts)
	if categoryCountsValue.CategoryScope != "Magic: The Gathering" {
		t.Fatalf("category release scope = %q, want Magic: The Gathering", categoryCountsValue.CategoryScope)
	}

	sets, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///1/sets"})
	if err != nil {
		t.Fatalf("ReadResource(category sets) error = %v", err)
	}
	setsValue := decodeResource[categorySetsOutput](t, sets)
	if got := len(setsValue.Sets); got != 2 {
		t.Fatalf("len(setsValue.Sets) = %d, want 2", got)
	}

	products, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///1/sets/100"})
	if err != nil {
		t.Fatalf("ReadResource(set products) error = %v", err)
	}
	productsValue := decodeResource[getSetProductsOutput](t, products)
	if productsValue.Pagination.Total != 3 || productsValue.Pagination.Returned != 3 || productsValue.Pagination.HasMore {
		t.Fatalf("resource products pagination = %+v, want total=3 returned=3 has_more=false", productsValue.Pagination)
	}

	pricing, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///1/sets/100/pricing"})
	if err != nil {
		t.Fatalf("ReadResource(set pricing) error = %v", err)
	}
	pricingValue := decodeResource[getSetPricingOutput](t, pricing)
	if got := len(pricingValue.Prices); got != 2 {
		t.Fatalf("len(pricingValue.Prices) = %d, want 2", got)
	}
	if pricingValue.UpdatedAt != "2026-03-27T12:00:00Z" {
		t.Fatalf("pricingValue.UpdatedAt = %q, want 2026-03-27T12:00:00Z", pricingValue.UpdatedAt)
	}
	if pricingValue.Prices[0].Product == nil || pricingValue.Prices[0].Product.Name != "Black Lotus" {
		t.Fatalf("pricingValue.Prices[0].Product = %+v, want Black Lotus summary", pricingValue.Prices[0].Product)
	}

	skus, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///1/sets/100/skus"})
	if err != nil {
		t.Fatalf("ReadResource(set skus) error = %v", err)
	}
	skusValue := decodeResource[getSetSKUsOutput](t, skus)
	if got := len(skusValue.Products); got != 2 {
		t.Fatalf("len(skusValue.Products) = %d, want 2", got)
	}
	if skusValue.UpdatedAt != "2026-03-27T13:00:00Z" {
		t.Fatalf("skusValue.UpdatedAt = %q, want 2026-03-27T13:00:00Z", skusValue.UpdatedAt)
	}
	if skusValue.Products[0].Product == nil || skusValue.Products[0].Product.Name != "Black Lotus" {
		t.Fatalf("skusValue.Products[0].Product = %+v, want Black Lotus summary", skusValue.Products[0].Product)
	}

	insights, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tcg:///1/sets/100/insights"})
	if err != nil {
		t.Fatalf("ReadResource(set insights) error = %v", err)
	}
	insightsValue := decodeResource[domain.SetInsights](t, insights)
	if insightsValue.Set.Name != "Alpha" {
		t.Fatalf("insightsValue.Set.Name = %q, want Alpha", insightsValue.Set.Name)
	}
	if got := len(insightsValue.HeuristicNotes); got != 3 {
		t.Fatalf("len(insightsValue.HeuristicNotes) = %d, want 3", got)
	}
	if insightsValue.ProductKindFilterApplied != domain.ProductKindFilterAll {
		t.Fatalf("insightsValue.ProductKindFilterApplied = %q, want %q", insightsValue.ProductKindFilterApplied, domain.ProductKindFilterAll)
	}
}

func TestPromptGetReturnsInstructions(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)
	ctx := context.Background()

	prompts := collectPrompts(t, ctx, clientSession)
	if got := len(prompts); got != 6 {
		t.Fatalf("len(prompts) = %d, want 6", got)
	}

	result, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "price-check",
		Arguments: map[string]string{
			"card_name": "Black Lotus",
			"game":      "mtg",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt(price-check) error = %v", err)
	}

	if got := len(result.Messages); got != 1 {
		t.Fatalf("len(result.Messages) = %d, want 1", got)
	}

	content, ok := result.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("prompt content type = %T, want *mcp.TextContent", result.Messages[0].Content)
	}

	for _, needle := range []string{"Black Lotus", "mtg", "search_sets", "get_set_products", "get_set_pricing", "updated_at", "tcg:///meta"} {
		if !strings.Contains(content.Text, needle) {
			t.Fatalf("prompt text missing %q: %s", needle, content.Text)
		}
	}

	expansionHistory, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "expansion-history",
		Arguments: map[string]string{
			"game":      "mtg",
			"year_from": "2001",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt(expansion-history) error = %v", err)
	}
	expansionContent, ok := expansionHistory.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expansion-history content type = %T, want *mcp.TextContent", expansionHistory.Messages[0].Content)
	}
	for _, needle := range []string{"summarize_release_counts", "2001", "mtg"} {
		if !strings.Contains(expansionContent.Text, needle) {
			t.Fatalf("expansion-history prompt missing %q: %s", needle, expansionContent.Text)
		}
	}

	setInsights, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "set-insights",
		Arguments: map[string]string{
			"set_name": "Alpha",
			"game":     "mtg",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt(set-insights) error = %v", err)
	}
	setInsightsContent, ok := setInsights.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("set-insights content type = %T, want *mcp.TextContent", setInsights.Messages[0].Content)
	}
	for _, needle := range []string{"analyze_set_insights", "product_kind_filter=single_like", "min_market_price=100", "top_n", "fields=[\"top_market_cards\"]", "omit fields"} {
		if !strings.Contains(setInsightsContent.Text, needle) {
			t.Fatalf("set-insights prompt missing %q: %s", needle, setInsightsContent.Text)
		}
	}

	valueDrivers, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "value-drivers",
		Arguments: map[string]string{
			"set_name": "Alpha",
			"game":     "mtg",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt(value-drivers) error = %v", err)
	}
	valueDriversContent, ok := valueDrivers.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("value-drivers content type = %T, want *mcp.TextContent", valueDrivers.Messages[0].Content)
	}
	for _, needle := range []string{"analyze_set_insights", "fields=[\"top_market_cards\",\"highest_value_rarity\",\"market_sum_estimate\"]", "artist", "illustration", "product_kind_filter=single_like", "min_market_price"} {
		if !strings.Contains(valueDriversContent.Text, needle) {
			t.Fatalf("value-drivers prompt missing %q: %s", needle, valueDriversContent.Text)
		}
	}
}

func newTestServer(t *testing.T) (*Server, *mcp.ClientSession, *fakeAPI) {
	t.Helper()
	return newTestServerWithAPI(t, newFakeAPI())
}

func newTestServerWithAPI(t *testing.T, api *fakeAPI) (*Server, *mcp.ClientSession, *fakeAPI) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resolver := catalog.NewResolver(api.categories, catalog.DefaultAliases())
	analyzer, err := analysis.New(analysis.Dependencies{
		API: api,
		Categories: func(context.Context) ([]domain.Category, error) {
			return append([]domain.Category(nil), api.categories...), nil
		},
	})
	if err != nil {
		t.Fatalf("analysis.New() error = %v", err)
	}

	srv, err := New(Dependencies{
		Logger:   logger,
		API:      api,
		Analyzer: analyzer,
		Resolver: resolver,
		PageSize: 2,
		Build: buildinfo.Info{
			Version: "v1.2.3-test",
			Commit:  "abc1234",
			Date:    "2026-03-27",
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := srv.raw.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("srv.raw.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = serverSession.Close()
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
	})

	return srv, clientSession, api
}

func newFakeAPI() *fakeAPI {
	categories := []domain.Category{
		{ID: 1, Name: "Magic", DisplayName: "Magic: The Gathering", ProductCount: 3, SetCount: 2, APIURL: "/1/sets"},
		{ID: 3, Name: "Pokemon", DisplayName: "Pokemon", ProductCount: 1, SetCount: 1, APIURL: "/3/sets"},
	}

	return &fakeAPI{
		meta: domain.Meta{
			Version:         "1.2.3",
			TotalCategories: 2,
			TotalSets:       3,
			TotalProducts:   4,
		},
		categories: categories,
		categorySets: map[int][]domain.SetSummary{
			1: {
				{ID: 100, CategoryID: 1, Name: "Alpha", Abbreviation: "LEA", PublishedOn: "2001-08-01", ProductCount: 3, SKUCount: 4},
				{ID: 101, CategoryID: 1, Name: "Beta", Abbreviation: "LEB", PublishedOn: "2003-08-01", ProductCount: 1, SKUCount: 1, IsSupplemental: true},
			},
			3: {
				{ID: 200, CategoryID: 3, Name: "Ruby", Abbreviation: "RBY", PublishedOn: "2001-01-01", ProductCount: 1, SKUCount: 1},
			},
		},
		searchSets: map[int][]domain.SetSummary{
			1: {
				{ID: 100, CategoryID: 1, Name: "Alpha", Abbreviation: "LEA", PublishedOn: "2001-08-01", ProductCount: 3, SKUCount: 4},
				{ID: 101, CategoryID: 1, Name: "Beta", Abbreviation: "LEB", PublishedOn: "2003-08-01", ProductCount: 1, SKUCount: 1, IsSupplemental: true},
			},
		},
		products: map[setKey][]domain.Product{
			{categoryID: 1, setID: 100}: {
				{ID: 10, SetID: 100, Name: "Black Lotus", SetName: "Alpha", Number: "233", Rarity: "Rare", Colors: []string{}, Finishes: []string{"nonfoil"}},
				{ID: 20, SetID: 100, Name: "Time Walk", SetName: "Alpha", Number: "1", Rarity: "Rare", Colors: []string{"U"}, Finishes: []string{"nonfoil"}},
				{ID: 30, SetID: 100, Name: "Mox Sapphire", SetName: "Alpha", Number: "265", Rarity: "Rare", Colors: []string{}, Finishes: []string{"nonfoil"}},
			},
		},
		pricing: map[setKey]domain.PricingSnapshot{
			{categoryID: 1, setID: 100}: {
				UpdatedAt: "2026-03-27T12:00:00Z",
				Prices: []domain.PricingResult{
					{ProductID: 10, Subtypes: map[string]domain.Price{"Normal": {Low: floatPtr(10000), Market: floatPtr(11000)}, "Foil": {Market: floatPtr(12000)}}, Manapool: map[string]float64{"Normal": 10500}},
					{ProductID: 20, Subtypes: map[string]domain.Price{"Normal": {Low: floatPtr(5000), Market: floatPtr(5500)}}, Manapool: map[string]float64{"Normal": 5300}},
				},
			},
		},
		skus: map[setKey]domain.SKUSnapshot{
			{categoryID: 1, setID: 100}: {
				UpdatedAt: "2026-03-27T13:00:00Z",
				Products: []domain.SKUResult{
					{ProductID: 10, SKUs: []domain.SKU{{ID: 1001, Condition: "NM", Variant: "N", Language: "en", Market: floatPtr(11000)}}},
					{ProductID: 20, SKUs: []domain.SKU{{ID: 2001, Condition: "NM", Variant: "N", Language: "en", Market: floatPtr(5500)}}},
				},
			},
		},
	}
}

func collectTools(t *testing.T, ctx context.Context, clientSession *mcp.ClientSession) []*mcp.Tool {
	t.Helper()

	var tools []*mcp.Tool
	for tool, err := range clientSession.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("clientSession.Tools() error = %v", err)
		}
		tools = append(tools, tool)
	}
	return tools
}

func collectResources(t *testing.T, ctx context.Context, clientSession *mcp.ClientSession) []*mcp.Resource {
	t.Helper()

	var resources []*mcp.Resource
	for resource, err := range clientSession.Resources(ctx, nil) {
		if err != nil {
			t.Fatalf("clientSession.Resources() error = %v", err)
		}
		resources = append(resources, resource)
	}
	return resources
}

func collectResourceTemplates(t *testing.T, ctx context.Context, clientSession *mcp.ClientSession) []*mcp.ResourceTemplate {
	t.Helper()

	var templates []*mcp.ResourceTemplate
	for template, err := range clientSession.ResourceTemplates(ctx, nil) {
		if err != nil {
			t.Fatalf("clientSession.ResourceTemplates() error = %v", err)
		}
		templates = append(templates, template)
	}
	return templates
}

func collectPrompts(t *testing.T, ctx context.Context, clientSession *mcp.ClientSession) []*mcp.Prompt {
	t.Helper()

	var prompts []*mcp.Prompt
	for prompt, err := range clientSession.Prompts(ctx, nil) {
		if err != nil {
			t.Fatalf("clientSession.Prompts() error = %v", err)
		}
		prompts = append(prompts, prompt)
	}
	return prompts
}

func decodeStructured[T any](t *testing.T, value any) T {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(structured content) error = %v", err)
	}

	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal(structured content) error = %v", err)
	}

	return out
}

func assertContainsKeys(t *testing.T, value map[string]any, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := value[key]; !ok {
			t.Fatalf("structured content missing key %q: %+v", key, value)
		}
	}
}

func assertOmitsKeys(t *testing.T, value map[string]any, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := value[key]; ok {
			t.Fatalf("structured content unexpectedly contains key %q: %+v", key, value)
		}
	}
}

func decodeResource[T any](t *testing.T, result *mcp.ReadResourceResult) T {
	t.Helper()

	if result == nil || len(result.Contents) == 0 {
		t.Fatal("resource result has no contents")
	}

	var out T
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &out); err != nil {
		t.Fatalf("json.Unmarshal(resource text) error = %v", err)
	}

	return out
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func floatPtr(value float64) *float64 {
	return &value
}
