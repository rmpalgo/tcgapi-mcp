package analysis

import (
	"context"
	"math"
	"testing"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

type setKey struct {
	categoryID int
	setID      int
}

type fakeAPI struct {
	meta         domain.Meta
	categories   []domain.Category
	categorySets map[int][]domain.SetSummary
	products     map[setKey][]domain.Product
	pricing      map[setKey]domain.PricingSnapshot
	skus         map[setKey]domain.SKUSnapshot
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

func (f *fakeAPI) SearchSets(context.Context, int, string) ([]domain.SetSummary, error) {
	return nil, nil
}

func (f *fakeAPI) SetProducts(_ context.Context, categoryID, setID int) ([]domain.Product, error) {
	return append([]domain.Product(nil), f.products[setKey{categoryID: categoryID, setID: setID}]...), nil
}

func (f *fakeAPI) SetPricing(_ context.Context, categoryID, setID int, _ *int) (domain.PricingSnapshot, error) {
	return f.pricing[setKey{categoryID: categoryID, setID: setID}], nil
}

func (f *fakeAPI) SetSKUs(_ context.Context, categoryID, setID int, _ *int) (domain.SKUSnapshot, error) {
	return f.skus[setKey{categoryID: categoryID, setID: setID}], nil
}

func TestSummarizeReleaseCountsFiltersByYearAndSupplemental(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{
		categories: []domain.Category{
			{ID: 1, Name: "Magic", DisplayName: "Magic: The Gathering"},
			{ID: 3, Name: "Pokemon", DisplayName: "Pokemon"},
		},
		categorySets: map[int][]domain.SetSummary{
			1: {
				{ID: 10, CategoryID: 1, Name: "Legacy", PublishedOn: "1999-10-01"},
				{ID: 11, CategoryID: 1, Name: "Odyssey", PublishedOn: "2001-10-01"},
				{ID: 12, CategoryID: 1, Name: "Deck Series", PublishedOn: "2003-06-01", IsSupplemental: true},
			},
			3: {
				{ID: 20, CategoryID: 3, Name: "Expedition", PublishedOn: "2001-09-15"},
				{ID: 21, CategoryID: 3, Name: "Ruby", PublishedOn: "2003-07-01"},
			},
		},
	}

	analyzer, err := New(Dependencies{
		API: api,
		Categories: func(context.Context) ([]domain.Category, error) {
			return api.Categories(context.Background())
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	yearTo := 2003
	summary, err := analyzer.SummarizeReleaseCounts(context.Background(), nil, 2001, &yearTo, false)
	if err != nil {
		t.Fatalf("SummarizeReleaseCounts() error = %v", err)
	}

	if summary.CategoryScope != "all categories" {
		t.Fatalf("CategoryScope = %q, want all categories", summary.CategoryScope)
	}
	if summary.TotalSets != 3 {
		t.Fatalf("TotalSets = %d, want 3", summary.TotalSets)
	}
	if summary.YearRange.From != 2001 || summary.YearRange.To != 2003 {
		t.Fatalf("YearRange = %+v, want 2001..2003", summary.YearRange)
	}
	if got := len(summary.CountsByYear); got != 3 {
		t.Fatalf("len(CountsByYear) = %d, want 3", got)
	}
	if summary.CountsByYear[0].Year != 2001 || summary.CountsByYear[0].SetCount != 2 {
		t.Fatalf("CountsByYear[0] = %+v, want 2001 => 2", summary.CountsByYear[0])
	}
	if summary.CountsByYear[1].Year != 2002 || summary.CountsByYear[1].SetCount != 0 {
		t.Fatalf("CountsByYear[1] = %+v, want 2002 => 0", summary.CountsByYear[1])
	}
	if summary.CountsByYear[2].Year != 2003 || summary.CountsByYear[2].SetCount != 1 {
		t.Fatalf("CountsByYear[2] = %+v, want 2003 => 1", summary.CountsByYear[2])
	}
}

func TestAnalyzeSetInsightsSummarizesRarityNumberingAndValue(t *testing.T) {
	t.Parallel()

	category := domain.Category{ID: 3, Name: "Pokemon", DisplayName: "Pokemon"}
	api := &fakeAPI{
		categorySets: map[int][]domain.SetSummary{
			3: {
				{ID: 100, CategoryID: 3, Name: "Ascended Heroes", Abbreviation: "ME", PublishedOn: "2026-03-01", ProductCount: 4, SKUCount: 12},
			},
		},
		products: map[setKey][]domain.Product{
			{categoryID: 3, setID: 100}: {
				{ID: 10, Name: "Mega Gengar ex", Number: "284/217", Rarity: "Special Illustration Rare"},
				{ID: 20, Name: "Mega Dragonite ex", Number: "290a", Rarity: "Special Illustration Rare"},
				{ID: 30, Name: "Mega Charizard Y ex", Number: "294", Rarity: "Mega Hyper Rare"},
				{ID: 40, Name: "Elite Trainer Box", Number: "", Rarity: ""},
			},
		},
		pricing: map[setKey]domain.PricingSnapshot{
			{categoryID: 3, setID: 100}: {
				UpdatedAt: "2026-03-27T08:04:10-04:00",
				Prices: []domain.PricingResult{
					{ProductID: 10, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(1000)}, "Holofoil": {Market: floatPtr(1075)}}},
					{ProductID: 20, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(682.19)}}},
					{ProductID: 30, Subtypes: map[string]domain.Price{"Normal": {Market: floatPtr(437.43)}}},
				},
			},
		},
		skus: map[setKey]domain.SKUSnapshot{
			{categoryID: 3, setID: 100}: {
				UpdatedAt: "2026-03-27T09:30:04-04:00",
			},
		},
	}

	analyzer, err := New(Dependencies{
		API: api,
		Categories: func(context.Context) ([]domain.Category, error) {
			return []domain.Category{category}, nil
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	insights, err := analyzer.AnalyzeSetInsights(context.Background(), category, 100, 2)
	if err != nil {
		t.Fatalf("AnalyzeSetInsights() error = %v", err)
	}

	if insights.ProductCountTotal != 4 {
		t.Fatalf("ProductCountTotal = %d, want 4", insights.ProductCountTotal)
	}
	if insights.NumberedCardLikeCount != 3 {
		t.Fatalf("NumberedCardLikeCount = %d, want 3", insights.NumberedCardLikeCount)
	}
	if insights.NumberingSummary.NumberedProducts != 3 || insights.NumberingSummary.UnnumberedProducts != 1 {
		t.Fatalf("NumberingSummary = %+v, want 3 numbered and 1 unnumbered", insights.NumberingSummary)
	}
	if !insights.NumberingSummary.HasSlashNumbers || !insights.NumberingSummary.HasLetterSuffixes {
		t.Fatalf("NumberingSummary flags = %+v, want slash and letter suffix detection", insights.NumberingSummary)
	}
	if len(insights.TopMarketCards) != 2 {
		t.Fatalf("len(TopMarketCards) = %d, want 2", len(insights.TopMarketCards))
	}
	if insights.TopMarketCards[0].ProductID != 10 || insights.TopMarketCards[0].Subtype != "Holofoil" {
		t.Fatalf("TopMarketCards[0] = %+v, want product 10 Holofoil", insights.TopMarketCards[0])
	}
	if insights.HighestValueRarity == nil || insights.HighestValueRarity.Rarity != "Special Illustration Rare" {
		t.Fatalf("HighestValueRarity = %+v, want Special Illustration Rare", insights.HighestValueRarity)
	}
	if !almostEqual(insights.MarketSumEstimate, 2194.62) {
		t.Fatalf("MarketSumEstimate = %f, want 2194.62", insights.MarketSumEstimate)
	}
	if insights.PricingUpdatedAt != "2026-03-27T08:04:10-04:00" {
		t.Fatalf("PricingUpdatedAt = %q, want pricing timestamp", insights.PricingUpdatedAt)
	}
	if insights.SKUUpdatedAt != "2026-03-27T09:30:04-04:00" {
		t.Fatalf("SKUUpdatedAt = %q, want SKU timestamp", insights.SKUUpdatedAt)
	}
	if got := len(insights.RarityBreakdown); got != 3 {
		t.Fatalf("len(RarityBreakdown) = %d, want 3", got)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func almostEqual(got, want float64) bool {
	return math.Abs(got-want) < 0.0001
}
