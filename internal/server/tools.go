package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rmpalgo/tcgapi-mcp/internal/analysis"
	"github.com/rmpalgo/tcgapi-mcp/internal/catalog"
	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

type listCategoriesOutput struct {
	Categories []domain.Category `json:"categories"`
}

type categorySetsOutput struct {
	CategoryID int                 `json:"category_id"`
	Sets       []domain.SetSummary `json:"sets"`
}

type searchSetsInput struct {
	Category string `json:"category" jsonschema:"game category name, alias, or numeric ID"`
	Query    string `json:"query" jsonschema:"set name or abbreviation to search for"`
}

type searchSetsOutput struct {
	Sets []domain.SetSummary `json:"sets"`
}

type getSetProductsInput struct {
	Category string `json:"category" jsonschema:"game category name, alias, or numeric ID"`
	SetID    int    `json:"set_id" jsonschema:"set ID from search_sets or the category sets resource"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max products to return; defaults to the server page size"`
	Offset   int    `json:"offset,omitempty" jsonschema:"zero-based offset into the set product list"`
}

type paginationOutput struct {
	Total    int  `json:"total"`
	Returned int  `json:"returned"`
	Offset   int  `json:"offset"`
	HasMore  bool `json:"has_more"`
}

type getSetProductsOutput struct {
	CategoryID int                         `json:"category_id"`
	SetID      int                         `json:"set_id"`
	Items      []catalog.NormalizedProduct `json:"items"`
	Pagination paginationOutput            `json:"pagination"`
}

type getSetPricingInput struct {
	Category  string `json:"category" jsonschema:"game category name, alias, or numeric ID"`
	SetID     int    `json:"set_id" jsonschema:"set ID from search_sets or the category sets resource"`
	ProductID *int   `json:"product_id,omitempty" jsonschema:"optional product ID to narrow the response to a single product"`
}

type getSetPricingOutput struct {
	CategoryID int                    `json:"category_id"`
	SetID      int                    `json:"set_id"`
	ProductID  *int                   `json:"product_id,omitempty"`
	UpdatedAt  string                 `json:"updated_at"`
	Prices     []domain.PricingResult `json:"prices"`
}

type getSetSKUsInput struct {
	Category  string `json:"category" jsonschema:"game category name, alias, or numeric ID"`
	SetID     int    `json:"set_id" jsonschema:"set ID from search_sets or the category sets resource"`
	ProductID *int   `json:"product_id,omitempty" jsonschema:"optional product ID to narrow the response to a single product"`
}

type getSetSKUsOutput struct {
	CategoryID int                `json:"category_id"`
	SetID      int                `json:"set_id"`
	ProductID  *int               `json:"product_id,omitempty"`
	UpdatedAt  string             `json:"updated_at"`
	Products   []domain.SKUResult `json:"products"`
}

type summarizeReleaseCountsInput struct {
	Category            string `json:"category,omitempty" jsonschema:"optional game category name, alias, or numeric ID"`
	YearFrom            int    `json:"year_from,omitempty" jsonschema:"inclusive starting year; defaults to 2001"`
	YearTo              *int   `json:"year_to,omitempty" jsonschema:"optional inclusive ending year"`
	IncludeSupplemental *bool  `json:"include_supplemental,omitempty" jsonschema:"whether to include supplemental sets; defaults to true"`
}

type analyzeSetInsightsInput struct {
	Category          string   `json:"category" jsonschema:"game category name, alias, or numeric ID"`
	SetID             int      `json:"set_id" jsonschema:"set ID from search_sets or the category sets resource"`
	TopN              int      `json:"top_n,omitempty" jsonschema:"how many top market cards to return; defaults to 10"`
	ProductKindFilter string   `json:"product_kind_filter,omitempty" jsonschema:"optional product-kind filter; supported values are all and single_like"`
	MinMarketPrice    *float64 `json:"min_market_price,omitempty" jsonschema:"optional minimum market price threshold for ranked outputs"`
	Fields            []string `json:"fields,omitempty" jsonschema:"optional analysis sections for token-efficient responses; supported values are numbering_summary, rarity_breakdown, top_market_cards, highest_value_rarity, market_sum_estimate, heuristic_notes"`
}

func (s *Server) registerTools() {
	s.tools = []*mcp.Tool{
		{
			Name:        "list_categories",
			Description: "List all game categories with IDs, names, and product/set counts. Use this first to resolve a game to category IDs and aliases.",
		},
		{
			Name:        "search_sets",
			Description: "Search sets by name or abbreviation within a game category. The category field accepts IDs, names, or aliases like 3, Pokemon, or mtg.",
		},
		{
			Name:        "get_set_products",
			Description: "Get products in a set. Results are normalized for LLM use and paginated server-side. The category field accepts IDs, names, or aliases.",
		},
		{
			Name:        "get_set_pricing",
			Description: "Get TCGPlayer and Manapool pricing for a set. Pass product_id to narrow the response to one product. The category field accepts IDs, names, or aliases.",
		},
		{
			Name:        "get_set_skus",
			Description: "Get SKU-level condition, variant, and language pricing for a set. Pass product_id to narrow the response to one product. The category field accepts IDs, names, or aliases.",
		},
		{
			Name:        "summarize_release_counts",
			Description: "Summarize how many sets were released by year after a chosen starting year, optionally scoped to one game category.",
		},
		{
			Name:        "analyze_set_insights",
			Description: "Derive set-level insights such as rarity counts, numbering patterns, top market cards, and a market-sum estimate for a set. Omit fields for the full payload, or pass fields to return only specific analysis sections while keeping the core metadata.",
		},
	}

	mcp.AddTool(s.raw, s.tools[0], func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listCategoriesOutput, error) {
		categories := s.resolver.Categories()
		if len(categories) == 0 {
			var err error
			categories, err = s.api.Categories(ctx)
			if err != nil {
				return nil, listCategoriesOutput{}, err
			}
		}

		return nil, listCategoriesOutput{
			Categories: categories,
		}, nil
	})

	mcp.AddTool(s.raw, s.tools[1], func(ctx context.Context, req *mcp.CallToolRequest, in searchSetsInput) (*mcp.CallToolResult, searchSetsOutput, error) {
		categoryID, err := s.resolver.ResolveCategoryID(in.Category)
		if err != nil {
			return nil, searchSetsOutput{}, err
		}

		sets, err := s.api.SearchSets(ctx, categoryID, in.Query)
		if err != nil {
			return nil, searchSetsOutput{}, err
		}

		return nil, searchSetsOutput{
			Sets: sets,
		}, nil
	})

	mcp.AddTool(s.raw, s.tools[2], func(ctx context.Context, req *mcp.CallToolRequest, in getSetProductsInput) (*mcp.CallToolResult, getSetProductsOutput, error) {
		categoryID, err := s.resolver.ResolveCategoryID(in.Category)
		if err != nil {
			return nil, getSetProductsOutput{}, err
		}

		products, err := s.api.SetProducts(ctx, categoryID, in.SetID)
		if err != nil {
			return nil, getSetProductsOutput{}, err
		}

		return nil, s.paginatedProductsOutput(categoryID, in.SetID, products, in.Limit, in.Offset), nil
	})

	mcp.AddTool(s.raw, s.tools[3], func(ctx context.Context, req *mcp.CallToolRequest, in getSetPricingInput) (*mcp.CallToolResult, getSetPricingOutput, error) {
		categoryID, err := s.resolver.ResolveCategoryID(in.Category)
		if err != nil {
			return nil, getSetPricingOutput{}, err
		}

		output, err := s.pricingOutput(ctx, categoryID, in.SetID, in.ProductID)
		if err != nil {
			return nil, getSetPricingOutput{}, err
		}
		return nil, output, nil
	})

	mcp.AddTool(s.raw, s.tools[4], func(ctx context.Context, req *mcp.CallToolRequest, in getSetSKUsInput) (*mcp.CallToolResult, getSetSKUsOutput, error) {
		categoryID, err := s.resolver.ResolveCategoryID(in.Category)
		if err != nil {
			return nil, getSetSKUsOutput{}, err
		}

		output, err := s.skuOutput(ctx, categoryID, in.SetID, in.ProductID)
		if err != nil {
			return nil, getSetSKUsOutput{}, err
		}
		return nil, output, nil
	})

	mcp.AddTool(s.raw, s.tools[5], func(ctx context.Context, req *mcp.CallToolRequest, in summarizeReleaseCountsInput) (*mcp.CallToolResult, domain.ReleaseCountsSummary, error) {
		var category *domain.Category
		if in.Category != "" {
			resolved, err := s.resolver.ResolveCategory(in.Category)
			if err != nil {
				return nil, domain.ReleaseCountsSummary{}, err
			}
			category = &resolved
		}

		includeSupplemental := true
		if in.IncludeSupplemental != nil {
			includeSupplemental = *in.IncludeSupplemental
		}

		summary, err := s.analyzer.SummarizeReleaseCounts(ctx, category, in.YearFrom, in.YearTo, includeSupplemental)
		if err != nil {
			return nil, domain.ReleaseCountsSummary{}, err
		}
		return nil, summary, nil
	})

	mcp.AddTool(s.raw, s.tools[6], func(ctx context.Context, req *mcp.CallToolRequest, in analyzeSetInsightsInput) (*mcp.CallToolResult, analyzeSetInsightsOutput, error) {
		category, err := s.resolver.ResolveCategory(in.Category)
		if err != nil {
			return nil, analyzeSetInsightsOutput{}, err
		}
		fieldSelection, err := parseAnalyzeSetInsightsFields(in.Fields)
		if err != nil {
			return nil, analyzeSetInsightsOutput{}, err
		}

		insights, err := s.analyzer.AnalyzeSetInsights(ctx, category, in.SetID, analysis.SetInsightsOptions{
			TopN:              in.TopN,
			ProductKindFilter: domain.ProductKindFilter(in.ProductKindFilter),
			MinMarketPrice:    in.MinMarketPrice,
		})
		if err != nil {
			return nil, analyzeSetInsightsOutput{}, err
		}
		return nil, newAnalyzeSetInsightsOutput(insights, fieldSelection), nil
	})
}

func (s *Server) paginatedProductsOutput(categoryID, setID int, products []domain.Product, limit, offset int) getSetProductsOutput {
	if limit <= 0 {
		limit = s.pageSize
	}

	paginated := catalog.PaginateProducts(products, limit, offset)
	items := make([]catalog.NormalizedProduct, 0, len(paginated.Items))
	for _, product := range paginated.Items {
		items = append(items, catalog.NormalizeProduct(product))
	}

	return getSetProductsOutput{
		CategoryID: categoryID,
		SetID:      setID,
		Items:      items,
		Pagination: paginationOutput{
			Total:    paginated.Pagination.Total,
			Returned: paginated.Pagination.Returned,
			Offset:   paginated.Pagination.Offset,
			HasMore:  paginated.Pagination.HasMore,
		},
	}
}

func fullProductsOutput(categoryID, setID int, products []domain.Product) getSetProductsOutput {
	paginated := catalog.PaginateProducts(products, len(products)+1, 0)
	items := make([]catalog.NormalizedProduct, 0, len(paginated.Items))
	for _, product := range paginated.Items {
		items = append(items, catalog.NormalizeProduct(product))
	}

	return getSetProductsOutput{
		CategoryID: categoryID,
		SetID:      setID,
		Items:      items,
		Pagination: paginationOutput{
			Total:    paginated.Pagination.Total,
			Returned: paginated.Pagination.Returned,
			Offset:   paginated.Pagination.Offset,
			HasMore:  paginated.Pagination.HasMore,
		},
	}
}
