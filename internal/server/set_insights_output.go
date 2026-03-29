package server

import (
	"fmt"
	"strings"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

const (
	analyzeSetInsightsFieldNumberingSummary = "numbering_summary"
	analyzeSetInsightsFieldRarityBreakdown  = "rarity_breakdown"
	analyzeSetInsightsFieldTopMarketCards   = "top_market_cards"
	analyzeSetInsightsFieldHighestValue     = "highest_value_rarity"
	analyzeSetInsightsFieldMarketSum        = "market_sum_estimate"
	analyzeSetInsightsFieldHeuristicNotes   = "heuristic_notes"
)

var supportedAnalyzeSetInsightsFields = []string{
	analyzeSetInsightsFieldNumberingSummary,
	analyzeSetInsightsFieldRarityBreakdown,
	analyzeSetInsightsFieldTopMarketCards,
	analyzeSetInsightsFieldHighestValue,
	analyzeSetInsightsFieldMarketSum,
	analyzeSetInsightsFieldHeuristicNotes,
}

type analyzeSetInsightsOutput struct {
	Set                      domain.SetMetadata         `json:"set"`
	ProductCountTotal        int                        `json:"product_count_total"`
	NumberedCardLikeCount    int                        `json:"numbered_card_like_count"`
	NumberingSummary         *domain.NumberingSummary   `json:"numbering_summary,omitempty"`
	RarityBreakdown          *[]domain.RarityBreakdown  `json:"rarity_breakdown,omitempty"`
	PricingUpdatedAt         string                     `json:"pricing_updated_at"`
	SKUUpdatedAt             string                     `json:"sku_updated_at"`
	TopMarketCards           *[]domain.TopMarketCard    `json:"top_market_cards,omitempty"`
	HighestValueRarity       *domain.HighestValueRarity `json:"highest_value_rarity,omitempty"`
	MarketSumEstimate        *float64                   `json:"market_sum_estimate,omitempty"`
	ProductKindFilterApplied domain.ProductKindFilter   `json:"product_kind_filter_applied"`
	MinMarketPriceApplied    *float64                   `json:"min_market_price_applied,omitempty"`
	HeuristicNotes           *[]string                  `json:"heuristic_notes,omitempty"`
}

type analyzeSetInsightsFieldSelection struct {
	includeAll         bool
	numberingSummary   bool
	rarityBreakdown    bool
	topMarketCards     bool
	highestValueRarity bool
	marketSumEstimate  bool
	heuristicNotes     bool
}

func newAnalyzeSetInsightsOutput(insights domain.SetInsights, selection analyzeSetInsightsFieldSelection) analyzeSetInsightsOutput {
	output := analyzeSetInsightsOutput{
		Set:                      insights.Set,
		ProductCountTotal:        insights.ProductCountTotal,
		NumberedCardLikeCount:    insights.NumberedCardLikeCount,
		PricingUpdatedAt:         insights.PricingUpdatedAt,
		SKUUpdatedAt:             insights.SKUUpdatedAt,
		ProductKindFilterApplied: insights.ProductKindFilterApplied,
		MinMarketPriceApplied:    cloneFloat64Pointer(insights.MinMarketPriceApplied),
	}

	if selection.includeAll || selection.numberingSummary {
		numbering := insights.NumberingSummary
		output.NumberingSummary = &numbering
	}
	if selection.includeAll || selection.rarityBreakdown {
		output.RarityBreakdown = cloneRarityBreakdown(insights.RarityBreakdown)
	}
	if selection.includeAll || selection.topMarketCards {
		output.TopMarketCards = cloneTopMarketCards(insights.TopMarketCards)
	}
	if selection.includeAll || selection.highestValueRarity {
		output.HighestValueRarity = cloneHighestValueRarity(insights.HighestValueRarity)
	}
	if selection.includeAll || selection.marketSumEstimate {
		output.MarketSumEstimate = float64Pointer(insights.MarketSumEstimate)
	}
	if selection.includeAll || selection.heuristicNotes {
		output.HeuristicNotes = cloneStrings(insights.HeuristicNotes)
	}

	return output
}

func parseAnalyzeSetInsightsFields(fields []string) (analyzeSetInsightsFieldSelection, error) {
	if fields == nil {
		return analyzeSetInsightsFieldSelection{includeAll: true}, nil
	}

	selection := analyzeSetInsightsFieldSelection{}
	seen := make(map[string]struct{}, len(fields))
	invalid := make([]string, 0)

	for _, raw := range fields {
		field := strings.TrimSpace(raw)
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}

		switch field {
		case analyzeSetInsightsFieldNumberingSummary:
			selection.numberingSummary = true
		case analyzeSetInsightsFieldRarityBreakdown:
			selection.rarityBreakdown = true
		case analyzeSetInsightsFieldTopMarketCards:
			selection.topMarketCards = true
		case analyzeSetInsightsFieldHighestValue:
			selection.highestValueRarity = true
		case analyzeSetInsightsFieldMarketSum:
			selection.marketSumEstimate = true
		case analyzeSetInsightsFieldHeuristicNotes:
			selection.heuristicNotes = true
		default:
			invalid = append(invalid, field)
		}
	}

	if len(invalid) > 0 {
		return analyzeSetInsightsFieldSelection{}, fmt.Errorf(
			"fields contains unsupported values [%s]; supported values are [%s]",
			strings.Join(invalid, ", "),
			strings.Join(supportedAnalyzeSetInsightsFields, ", "),
		)
	}

	return selection, nil
}

func cloneRarityBreakdown(values []domain.RarityBreakdown) *[]domain.RarityBreakdown {
	cloned := make([]domain.RarityBreakdown, 0, len(values))
	cloned = append(cloned, values...)
	return &cloned
}

func cloneTopMarketCards(values []domain.TopMarketCard) *[]domain.TopMarketCard {
	cloned := make([]domain.TopMarketCard, 0, len(values))
	cloned = append(cloned, values...)
	return &cloned
}

func cloneHighestValueRarity(value *domain.HighestValueRarity) *domain.HighestValueRarity {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneStrings(values []string) *[]string {
	cloned := make([]string, 0, len(values))
	cloned = append(cloned, values...)
	return &cloned
}

func cloneFloat64Pointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func float64Pointer(value float64) *float64 {
	return &value
}
