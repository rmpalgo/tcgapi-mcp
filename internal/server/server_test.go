package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
	if got := len(tools); got != 5 {
		t.Fatalf("len(tools) = %d, want 5", got)
	}

	resources := collectResources(t, ctx, clientSession)
	if got := len(resources); got != 2 {
		t.Fatalf("len(resources) = %d, want 2", got)
	}

	templates := collectResourceTemplates(t, ctx, clientSession)
	if got := len(templates); got != 4 {
		t.Fatalf("len(resource templates) = %d, want 4", got)
	}

	prompts := collectPrompts(t, ctx, clientSession)
	if got := len(prompts); got != 3 {
		t.Fatalf("len(prompts) = %d, want 3", got)
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
	if len(api.skuCalls) != 1 || api.skuCalls[0].productID == nil || *api.skuCalls[0].productID != productID {
		t.Fatalf("sku calls = %+v, want product_id=%d", api.skuCalls, productID)
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
}

func TestPromptGetReturnsInstructions(t *testing.T) {
	t.Parallel()

	_, clientSession, _ := newTestServer(t)
	ctx := context.Background()

	prompts := collectPrompts(t, ctx, clientSession)
	if got := len(prompts); got != 3 {
		t.Fatalf("len(prompts) = %d, want 3", got)
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
}

func newTestServer(t *testing.T) (*Server, *mcp.ClientSession, *fakeAPI) {
	t.Helper()
	return newTestServerWithAPI(t, newFakeAPI())
}

func newTestServerWithAPI(t *testing.T, api *fakeAPI) (*Server, *mcp.ClientSession, *fakeAPI) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	resolver := catalog.NewResolver(api.categories, catalog.DefaultAliases())

	srv, err := New(Dependencies{
		Logger:   logger,
		API:      api,
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
				{ID: 100, CategoryID: 1, Name: "Alpha", Abbreviation: "LEA", ProductCount: 3, SKUCount: 4},
				{ID: 101, CategoryID: 1, Name: "Beta", Abbreviation: "LEB", ProductCount: 1, SKUCount: 1},
			},
		},
		searchSets: map[int][]domain.SetSummary{
			1: {
				{ID: 100, CategoryID: 1, Name: "Alpha", Abbreviation: "LEA", ProductCount: 3, SKUCount: 4},
				{ID: 101, CategoryID: 1, Name: "Beta", Abbreviation: "LEB", ProductCount: 1, SKUCount: 1},
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
					{ProductID: 10, Subtypes: map[string]domain.Price{"Normal": {Low: floatPtr(10000), Market: floatPtr(11000)}}, Manapool: map[string]float64{"Normal": 10500}},
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
