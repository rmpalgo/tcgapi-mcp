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
	}

	s.raw.AddPrompt(s.prompts[0], s.priceCheckPrompt)
	s.raw.AddPrompt(s.prompts[1], s.setOverviewPrompt)
	s.raw.AddPrompt(s.prompts[2], s.compareVariantsPrompt)
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
