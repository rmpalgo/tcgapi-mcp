package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerPrompts() {
	s.prompts = []*mcp.Prompt{
		{
			Name:        "price-check",
			Description: "Resolve a game, find a set, locate a product, fetch pricing, and summarize subtype prices.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "card_name",
					Description: "Card name to price-check.",
					Required:    true,
				},
				{
					Name:        "game",
					Description: "Optional game name or category alias, such as Pokemon or mtg.",
				},
			},
		},
		{
			Name:        "set-overview",
			Description: "Resolve a game, find a set, fetch products, inspect rarity and pricing, and present a set guide.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "set_name",
					Description: "Set name to inspect.",
					Required:    true,
				},
				{
					Name:        "game",
					Description: "Game name or category alias, such as Pokemon or mtg.",
					Required:    true,
				},
			},
		},
		{
			Name:        "compare-variants",
			Description: "Locate a product, fetch SKU data, and summarize condition, variant, and language prices.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "card_name",
					Description: "Card name to compare.",
					Required:    true,
				},
				{
					Name:        "game",
					Description: "Game name or category alias, such as Pokemon or mtg.",
					Required:    true,
				},
			},
		},
		{
			Name:        "expansion-history",
			Description: "Summarize how many sets were released after a given year, optionally scoped to one game category.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "game",
					Description: "Optional game name or category alias, such as Pokemon or mtg.",
				},
				{
					Name:        "year_from",
					Description: "Optional inclusive starting year. Defaults to 2001.",
				},
			},
		},
		{
			Name:        "set-insights",
			Description: "Build a deterministic set summary covering size, numbering, rarities, and top market cards.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "set_name",
					Description: "Set name to inspect.",
					Required:    true,
				},
				{
					Name:        "game",
					Description: "Game name or category alias, such as Pokemon or mtg.",
					Required:    true,
				},
			},
		},
		{
			Name:        "value-drivers",
			Description: "Explain what current set data suggests about value drivers, while clearly calling out unsupported factors.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "set_name",
					Description: "Set name to inspect.",
					Required:    true,
				},
				{
					Name:        "game",
					Description: "Game name or category alias, such as Pokemon or mtg.",
					Required:    true,
				},
			},
		},
	}

	s.raw.AddPrompt(s.prompts[0], s.priceCheckPrompt)
	s.raw.AddPrompt(s.prompts[1], s.setOverviewPrompt)
	s.raw.AddPrompt(s.prompts[2], s.compareVariantsPrompt)
	s.raw.AddPrompt(s.prompts[3], s.expansionHistoryPrompt)
	s.raw.AddPrompt(s.prompts[4], s.setInsightsPrompt)
	s.raw.AddPrompt(s.prompts[5], s.valueDriversPrompt)
}

func (s *Server) priceCheckPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	_ = ctx

	cardName := promptArg(req, "card_name")
	game := promptArg(req, "game")

	instructions := []string{
		fmt.Sprintf("Use the tcgapi-mcp tools to price-check the card %q.", cardName),
	}
	if game == "" {
		instructions = append(instructions, "1. Call list_categories to find the most likely game category for the user request.")
	} else {
		instructions = append(instructions, fmt.Sprintf("1. Treat %q as the target game and resolve it to a category using list_categories only if you need confirmation.", game))
	}
	instructions = append(instructions,
		"2. Call search_sets with the chosen category and a focused set query.",
		fmt.Sprintf("3. Call get_set_products to find the exact product that best matches %q.", cardName),
		"4. Call get_set_pricing with category, set_id, and product_id for the chosen product.",
		"5. Cite updated_at from the pricing response before making freshness claims such as today, this week, or recently.",
		"6. If the user asks about overall API freshness instead of this specific product response, read tcg:///meta.",
		"7. Summarize subtype low and market prices, and clearly note ambiguity if multiple products are plausible matches.",
	)

	return promptResult("price-check", strings.Join(instructions, "\n")), nil
}

func (s *Server) setOverviewPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	_ = ctx

	setName := promptArg(req, "set_name")
	game := promptArg(req, "game")

	text := strings.Join([]string{
		fmt.Sprintf("Use the tcgapi-mcp tools to build a set overview for %q in %q.", setName, game),
		"1. Resolve the game with list_categories if needed.",
		fmt.Sprintf("2. Call search_sets using the game and the set query %q.", setName),
		"3. Pick the best matching set and call get_set_products.",
		"4. Review rarity, numbering, and notable product traits from the product list.",
		"5. Call get_set_pricing for the chosen set and use updated_at if you mention recency.",
		"6. Read tcg:///meta if you need overall API freshness context beyond the set pricing response.",
		"7. Present the result as a concise set guide with any assumptions called out explicitly.",
	}, "\n")

	return promptResult("set-overview", text), nil
}

func (s *Server) compareVariantsPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	_ = ctx

	cardName := promptArg(req, "card_name")
	game := promptArg(req, "game")

	text := strings.Join([]string{
		fmt.Sprintf("Use the tcgapi-mcp tools to compare variants for %q in %q.", cardName, game),
		"1. Resolve the game with list_categories if needed.",
		"2. Use search_sets to locate the most likely set.",
		fmt.Sprintf("3. Use get_set_products to find the exact product match for %q.", cardName),
		"4. Call get_set_skus with category, set_id, and product_id.",
		"5. Cite updated_at from the SKU response before making freshness claims.",
		"6. If the user asks about overall API freshness instead of this SKU snapshot, read tcg:///meta.",
		"7. Summarize condition, variant, and language prices in a compact comparison table and note missing markets.",
	}, "\n")

	return promptResult("compare-variants", text), nil
}

func (s *Server) expansionHistoryPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	_ = ctx

	game := promptArg(req, "game")
	yearFrom := promptArg(req, "year_from")
	if yearFrom == "" {
		yearFrom = "2001"
	}

	lines := []string{
		fmt.Sprintf("Use the tcgapi-mcp analytics capabilities to summarize set releases from %s onward.", yearFrom),
	}
	if game == "" {
		lines = append(lines,
			"1. Read tcg:///analytics/releases-by-year for the default after-2000 view across all categories, or call summarize_release_counts if you need a custom year range.",
		)
	} else {
		lines = append(lines,
			fmt.Sprintf("1. Resolve %q to a category with list_categories if needed.", game),
			"2. Call summarize_release_counts with the chosen category and requested year range.",
		)
	}
	lines = append(lines,
		"3. Report the total number of sets and the per-year breakdown.",
		"4. If you mention heuristics like vintage versus modern, frame them as a simple pre-2001 versus 2001+ distinction rather than an official taxonomy.",
	)

	return promptResult("expansion-history", strings.Join(lines, "\n")), nil
}

func (s *Server) setInsightsPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	_ = ctx

	setName := promptArg(req, "set_name")
	game := promptArg(req, "game")

	text := strings.Join([]string{
		fmt.Sprintf("Use the tcgapi-mcp analytics capabilities to summarize set insights for %q in %q.", setName, game),
		"1. Resolve the game with list_categories if needed.",
		fmt.Sprintf("2. Use search_sets with %q to find the best matching set.", setName),
		"3. Use analyze_set_insights for the selected set.",
		"4. For singles-only style questions, call analyze_set_insights with product_kind_filter=single_like.",
		"5. For threshold questions like over $100, also pass min_market_price=100 and raise top_n high enough to capture the full list.",
		"6. Summarize set size, numbering patterns, rarity breakdown, top market cards, and highest_value_rarity.",
		"7. Treat numbered_card_like_count, product_kind, and market_sum_estimate as heuristics and label them accordingly.",
		"8. Cite pricing_updated_at and sku_updated_at when making recency-sensitive claims.",
	}, "\n")

	return promptResult("set-insights", text), nil
}

func (s *Server) valueDriversPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	_ = ctx

	setName := promptArg(req, "set_name")
	game := promptArg(req, "game")

	text := strings.Join([]string{
		fmt.Sprintf("Use the tcgapi-mcp analytics capabilities to explain value drivers for %q in %q.", setName, game),
		"1. Resolve the game with list_categories if needed.",
		fmt.Sprintf("2. Use search_sets with %q to find the target set.", setName),
		"3. Use analyze_set_insights for the selected set.",
		"4. If the user wants singles rather than sealed products, call analyze_set_insights with product_kind_filter=single_like.",
		"5. If the user wants a price threshold, add min_market_price and increase top_n as needed.",
		"6. Explain supported drivers using current data: rarity, numbering patterns, variant subtype pricing, and highest-value products.",
		"7. Explicitly say that artist, illustration style, pull-rate math, booster-pack slotting, and sealed-versus-single economics are not directly modeled by the current API.",
		"8. Treat product_kind and market_sum_estimate as heuristics, not authoritative upstream classifications or valuations.",
	}, "\n")

	return promptResult("value-drivers", text), nil
}

func promptResult(description, text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []*mcp.PromptMessage{
			{
				Role:    mcp.Role("user"),
				Content: &mcp.TextContent{Text: text},
			},
		},
	}
}

func promptArg(req *mcp.GetPromptRequest, key string) string {
	if req == nil || req.Params == nil || req.Params.Arguments == nil {
		return ""
	}
	return strings.TrimSpace(req.Params.Arguments[key])
}
