package analysis

var setInsightsHeuristicNotes = []string{
	"numbered_card_like_count is a heuristic that treats products with a collector number and/or rarity as card-like products.",
	"market_sum_estimate is derived from the highest available TCGPlayer market subtype per product and is not an official set valuation.",
	"product_kind is a heuristic classification derived from product number, rarity, and normalized product names; it is not an upstream taxonomy.",
}

func SetInsightsHeuristicNotes() []string {
	return append([]string(nil), setInsightsHeuristicNotes...)
}
