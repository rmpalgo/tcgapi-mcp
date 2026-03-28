package analysis

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi"
)

type Service interface {
	SummarizeReleaseCounts(context.Context, *domain.Category, int, *int, bool) (domain.ReleaseCountsSummary, error)
	AnalyzeSetInsights(context.Context, domain.Category, int, SetInsightsOptions) (domain.SetInsights, error)
}

type CategoryProvider func(context.Context) ([]domain.Category, error)

type SetInsightsOptions struct {
	TopN              int
	ProductKindFilter domain.ProductKindFilter
	MinMarketPrice    *float64
}

type Dependencies struct {
	API        tcgapi.API
	Categories CategoryProvider
}

type Analyzer struct {
	api        tcgapi.API
	categories CategoryProvider
}

var _ Service = (*Analyzer)(nil)

func New(d Dependencies) (*Analyzer, error) {
	if d.API == nil {
		return nil, fmt.Errorf("analysis API dependency is required")
	}
	if d.Categories == nil {
		return nil, fmt.Errorf("analysis categories provider is required")
	}

	return &Analyzer{
		api:        d.API,
		categories: d.Categories,
	}, nil
}

func (a *Analyzer) SummarizeReleaseCounts(ctx context.Context, category *domain.Category, yearFrom int, yearTo *int, includeSupplemental bool) (domain.ReleaseCountsSummary, error) {
	if yearFrom <= 0 {
		yearFrom = 2001
	}
	if yearTo != nil && *yearTo < yearFrom {
		return domain.ReleaseCountsSummary{}, fmt.Errorf("year_to must be >= year_from")
	}

	var categories []domain.Category
	if category != nil {
		categories = []domain.Category{*category}
	} else {
		var err error
		categories, err = a.categories(ctx)
		if err != nil {
			return domain.ReleaseCountsSummary{}, fmt.Errorf("load categories: %w", err)
		}
	}

	counts := make(map[int]int)
	maxYear := yearFrom
	total := 0

	for _, current := range categories {
		sets, err := a.api.CategorySets(ctx, current.ID)
		if err != nil {
			return domain.ReleaseCountsSummary{}, fmt.Errorf("load category %d sets: %w", current.ID, err)
		}

		for _, set := range sets {
			if !includeSupplemental && set.IsSupplemental {
				continue
			}

			year, ok := publishedYear(set.PublishedOn)
			if !ok || year < yearFrom {
				continue
			}
			if yearTo != nil && year > *yearTo {
				continue
			}

			counts[year]++
			total++
			if year > maxYear {
				maxYear = year
			}
		}
	}

	if yearTo != nil {
		maxYear = *yearTo
	}

	byYear := make([]domain.ReleaseYearCount, 0, maxYear-yearFrom+1)
	for year := yearFrom; year <= maxYear; year++ {
		byYear = append(byYear, domain.ReleaseYearCount{
			Year:     year,
			SetCount: counts[year],
		})
	}

	scope := "all categories"
	if category != nil {
		scope = categoryLabel(*category)
	}

	return domain.ReleaseCountsSummary{
		CategoryScope:        scope,
		YearRange:            domain.YearRange{From: yearFrom, To: maxYear},
		SupplementalIncluded: includeSupplemental,
		TotalSets:            total,
		CountsByYear:         byYear,
	}, nil
}

func (a *Analyzer) AnalyzeSetInsights(ctx context.Context, category domain.Category, setID int, options SetInsightsOptions) (domain.SetInsights, error) {
	options, err := normalizeSetInsightsOptions(options)
	if err != nil {
		return domain.SetInsights{}, err
	}

	sets, err := a.api.CategorySets(ctx, category.ID)
	if err != nil {
		return domain.SetInsights{}, fmt.Errorf("load category sets: %w", err)
	}
	setSummary, ok := findSetSummary(sets, setID)
	if !ok {
		return domain.SetInsights{}, fmt.Errorf("set %d not found in category %d", setID, category.ID)
	}

	products, err := a.api.SetProducts(ctx, category.ID, setID)
	if err != nil {
		return domain.SetInsights{}, fmt.Errorf("load set products: %w", err)
	}
	pricing, err := a.api.SetPricing(ctx, category.ID, setID, nil)
	if err != nil {
		return domain.SetInsights{}, fmt.Errorf("load set pricing: %w", err)
	}
	skus, err := a.api.SetSKUs(ctx, category.ID, setID, nil)
	if err != nil {
		return domain.SetInsights{}, fmt.Errorf("load set skus: %w", err)
	}

	perProductValue := buildProductValueMap(pricing)
	productKinds := classifyProducts(products)
	filteredProducts := filterInsightProducts(products, perProductValue, productKinds, options)
	rarityBreakdown := summarizeRarities(filteredProducts, perProductValue)
	numbering := summarizeNumbering(products)
	topCards, marketSum := buildTopCards(filteredProducts, perProductValue, productKinds, options.TopN)
	highestRarity := highestValueRarity(topCards)

	return domain.SetInsights{
		Set: domain.SetMetadata{
			ID:             setSummary.ID,
			CategoryID:     setSummary.CategoryID,
			Name:           setSummary.Name,
			Abbreviation:   setSummary.Abbreviation,
			IsSupplemental: setSummary.IsSupplemental,
			PublishedOn:    setSummary.PublishedOn,
			ProductCount:   setSummary.ProductCount,
			SKUCount:       setSummary.SKUCount,
		},
		ProductCountTotal:        len(products),
		NumberedCardLikeCount:    countNumberedCardLike(products),
		NumberingSummary:         numbering,
		RarityBreakdown:          rarityBreakdown,
		PricingUpdatedAt:         pricing.UpdatedAt,
		SKUUpdatedAt:             skus.UpdatedAt,
		TopMarketCards:           topCards,
		HighestValueRarity:       highestRarity,
		MarketSumEstimate:        marketSum,
		ProductKindFilterApplied: options.ProductKindFilter,
		MinMarketPriceApplied:    cloneFloat64(options.MinMarketPrice),
		HeuristicNotes: []string{
			"numbered_card_like_count is a heuristic that treats products with a collector number and/or rarity as card-like products.",
			"market_sum_estimate is derived from the highest available TCGPlayer market subtype per product and is not an official set valuation.",
			"product_kind is a heuristic classification derived from product number, rarity, and normalized product names; it is not an upstream taxonomy.",
		},
	}, nil
}

func normalizeSetInsightsOptions(options SetInsightsOptions) (SetInsightsOptions, error) {
	if options.TopN <= 0 {
		options.TopN = 10
	}
	if options.ProductKindFilter == "" {
		options.ProductKindFilter = domain.ProductKindFilterAll
	}
	switch options.ProductKindFilter {
	case domain.ProductKindFilterAll, domain.ProductKindFilterSingleLike:
	default:
		return SetInsightsOptions{}, fmt.Errorf("product_kind_filter must be one of %q or %q", domain.ProductKindFilterAll, domain.ProductKindFilterSingleLike)
	}
	if options.MinMarketPrice != nil && *options.MinMarketPrice < 0 {
		return SetInsightsOptions{}, fmt.Errorf("min_market_price must be >= 0")
	}
	return options, nil
}

func publishedYear(value string) (int, bool) {
	if strings.TrimSpace(value) == "" {
		return 0, false
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return 0, false
	}
	return t.Year(), true
}

func categoryLabel(category domain.Category) string {
	if strings.TrimSpace(category.DisplayName) != "" {
		return category.DisplayName
	}
	return category.Name
}

func findSetSummary(sets []domain.SetSummary, setID int) (domain.SetSummary, bool) {
	for _, set := range sets {
		if set.ID == setID {
			return set, true
		}
	}
	return domain.SetSummary{}, false
}

type productValue struct {
	Subtype string
	Market  float64
}

func buildProductValueMap(pricing domain.PricingSnapshot) map[int]productValue {
	values := make(map[int]productValue, len(pricing.Prices))
	for _, result := range pricing.Prices {
		subtype, market, ok := highestMarketSubtype(result)
		if !ok {
			continue
		}
		values[result.ProductID] = productValue{
			Subtype: subtype,
			Market:  market,
		}
	}
	return values
}

func highestMarketSubtype(result domain.PricingResult) (string, float64, bool) {
	if len(result.Subtypes) == 0 {
		return "", 0, false
	}

	keys := make([]string, 0, len(result.Subtypes))
	for subtype := range result.Subtypes {
		keys = append(keys, subtype)
	}
	sort.Strings(keys)

	bestSubtype := ""
	bestMarket := 0.0
	found := false

	for _, subtype := range keys {
		price := result.Subtypes[subtype]
		if price.Market == nil {
			continue
		}
		if !found || *price.Market > bestMarket {
			bestSubtype = subtype
			bestMarket = *price.Market
			found = true
		}
	}

	return bestSubtype, bestMarket, found
}

func summarizeRarities(products []domain.Product, values map[int]productValue) []domain.RarityBreakdown {
	type bucket struct {
		productCount     int
		marketCardCount  int
		totalMarketValue float64
		highestMarket    *float64
	}

	buckets := make(map[string]*bucket)
	for _, product := range products {
		rarity := normalizedRarity(product.Rarity)
		current := buckets[rarity]
		if current == nil {
			current = &bucket{}
			buckets[rarity] = current
		}
		current.productCount++

		if value, ok := values[product.ID]; ok {
			current.marketCardCount++
			current.totalMarketValue += value.Market
			if current.highestMarket == nil || value.Market > *current.highestMarket {
				highest := value.Market
				current.highestMarket = &highest
			}
		}
	}

	rarities := make([]string, 0, len(buckets))
	for rarity := range buckets {
		rarities = append(rarities, rarity)
	}
	sort.Strings(rarities)

	out := make([]domain.RarityBreakdown, 0, len(rarities))
	for _, rarity := range rarities {
		current := buckets[rarity]
		out = append(out, domain.RarityBreakdown{
			Rarity:             rarity,
			ProductCount:       current.productCount,
			MarketCardCount:    current.marketCardCount,
			TotalMarketValue:   current.totalMarketValue,
			HighestMarketValue: cloneFloat64(current.highestMarket),
		})
	}
	return out
}

func summarizeNumbering(products []domain.Product) domain.NumberingSummary {
	unique := make(map[string]struct{})
	samples := make([]string, 0, 10)
	numbered := 0
	hasSlash := false
	hasLetterSuffix := false

	for _, product := range products {
		number := strings.TrimSpace(product.Number)
		if number == "" {
			continue
		}
		numbered++
		if strings.Contains(number, "/") {
			hasSlash = true
		}
		if containsLetter(number) {
			hasLetterSuffix = true
		}
		if _, ok := unique[number]; ok {
			continue
		}
		unique[number] = struct{}{}
		samples = append(samples, number)
	}

	sort.Strings(samples)
	if len(samples) > 10 {
		samples = samples[:10]
	}

	return domain.NumberingSummary{
		NumberedProducts:   numbered,
		UnnumberedProducts: len(products) - numbered,
		SampleNumbers:      samples,
		HasSlashNumbers:    hasSlash,
		HasLetterSuffixes:  hasLetterSuffix,
	}
}

func buildTopCards(products []domain.Product, values map[int]productValue, productKinds map[int]domain.ProductKind, topN int) ([]domain.TopMarketCard, float64) {
	type valuedProduct struct {
		product domain.Product
		value   productValue
	}

	valued := make([]valuedProduct, 0, len(values))
	sum := 0.0

	for _, product := range products {
		value, ok := values[product.ID]
		if !ok {
			continue
		}
		valued = append(valued, valuedProduct{product: product, value: value})
		sum += value.Market
	}

	sort.Slice(valued, func(i, j int) bool {
		if valued[i].value.Market != valued[j].value.Market {
			return valued[i].value.Market > valued[j].value.Market
		}
		if valued[i].product.Name != valued[j].product.Name {
			return valued[i].product.Name < valued[j].product.Name
		}
		return valued[i].product.ID < valued[j].product.ID
	})

	if topN > len(valued) {
		topN = len(valued)
	}

	out := make([]domain.TopMarketCard, 0, topN)
	for _, current := range valued[:topN] {
		out = append(out, domain.TopMarketCard{
			ProductID:   current.product.ID,
			Name:        current.product.Name,
			Number:      current.product.Number,
			Rarity:      normalizedRarity(current.product.Rarity),
			ProductKind: productKinds[current.product.ID],
			Subtype:     current.value.Subtype,
			MarketPrice: current.value.Market,
		})
	}

	return out, sum
}

func highestValueRarity(cards []domain.TopMarketCard) *domain.HighestValueRarity {
	if len(cards) == 0 {
		return nil
	}

	top := cards[0]
	return &domain.HighestValueRarity{
		Rarity:      normalizedRarity(top.Rarity),
		ProductID:   top.ProductID,
		ProductName: top.Name,
		Number:      top.Number,
		ProductKind: top.ProductKind,
		Subtype:     top.Subtype,
		MarketPrice: top.MarketPrice,
	}
}

var sealedProductPhrases = []string{
	"booster box",
	"booster bundle",
	"booster pack",
	"pack art bundle",
	"build and battle",
	"elite trainer box",
	"blister",
	"tin",
	"display",
	"sleeved booster",
	"half booster",
	"case",
}

func classifyProducts(products []domain.Product) map[int]domain.ProductKind {
	kinds := make(map[int]domain.ProductKind, len(products))
	for _, product := range products {
		kinds[product.ID] = classifyProduct(product)
	}
	return kinds
}

func classifyProduct(product domain.Product) domain.ProductKind {
	rarity := strings.TrimSpace(product.Rarity)
	name := normalizedProductName(product)

	if strings.EqualFold(rarity, "Code Card") || strings.Contains(name, "code card") {
		return domain.ProductKindCodeCard
	}
	if strings.TrimSpace(product.Number) == "" && containsAnyPhrase(name, sealedProductPhrases) {
		return domain.ProductKindSealedLike
	}
	if strings.TrimSpace(product.Number) != "" || (rarity != "" && !strings.EqualFold(rarity, "Code Card")) {
		return domain.ProductKindSingleLike
	}
	return domain.ProductKindUnknown
}

func normalizedProductName(product domain.Product) string {
	value := product.CleanName
	if strings.TrimSpace(value) == "" {
		value = product.Name
	}

	var builder strings.Builder
	builder.Grow(len(value))
	lastSpace := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastSpace = false
			continue
		}
		if lastSpace {
			continue
		}
		builder.WriteByte(' ')
		lastSpace = true
	}

	return strings.TrimSpace(builder.String())
}

func containsAnyPhrase(value string, phrases []string) bool {
	padded := " " + value + " "
	for _, phrase := range phrases {
		if strings.Contains(padded, " "+phrase+" ") {
			return true
		}
	}
	return false
}

func filterInsightProducts(products []domain.Product, values map[int]productValue, productKinds map[int]domain.ProductKind, options SetInsightsOptions) []domain.Product {
	filtered := make([]domain.Product, 0, len(products))
	for _, product := range products {
		if !matchesProductKindFilter(productKinds[product.ID], options.ProductKindFilter) {
			continue
		}
		if options.MinMarketPrice != nil {
			value, ok := values[product.ID]
			if !ok || value.Market < *options.MinMarketPrice {
				continue
			}
		}
		filtered = append(filtered, product)
	}
	return filtered
}

func matchesProductKindFilter(kind domain.ProductKind, filter domain.ProductKindFilter) bool {
	switch filter {
	case domain.ProductKindFilterAll:
		return true
	case domain.ProductKindFilterSingleLike:
		return kind == domain.ProductKindSingleLike
	default:
		return true
	}
}

func countNumberedCardLike(products []domain.Product) int {
	total := 0
	for _, product := range products {
		if strings.TrimSpace(product.Number) != "" || strings.TrimSpace(product.Rarity) != "" {
			total++
		}
	}
	return total
}

func normalizedRarity(rarity string) string {
	if strings.TrimSpace(rarity) == "" {
		return "Unknown"
	}
	return rarity
}

func containsLetter(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
