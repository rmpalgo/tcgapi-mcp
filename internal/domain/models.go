package domain

type Meta struct {
	LastUpdated     string
	PricingUpdated  string
	TotalCategories int
	TotalSets       int
	TotalProducts   int
	Version         string
	Documentation   string
}

type Category struct {
	ID           int
	Name         string
	DisplayName  string
	ProductCount int
	SetCount     int
	APIURL       string
}

type SetSummary struct {
	ID             int
	CategoryID     int
	Name           string
	Abbreviation   string
	IsSupplemental bool
	PublishedOn    string
	ProductCount   int
	SKUCount       int
	APIURL         string
	PricingURL     string
	SKUsURL        string
}

type Product struct {
	ID                 int
	SetID              int
	Name               string
	CleanName          string
	SetName            string
	SetAbbreviation    string
	Number             string
	Rarity             string
	ImageURL           string
	TCGPlayerURL       string
	ManapoolURL        string
	ScryfallID         string
	MtgjsonUUID        string
	CardMarketID       *int
	CardTraderID       *int
	Colors             []string
	ManaValue          *float64
	Finishes           []string
	IsPresale          bool
	PresaleReleaseDate string
	PresaleNote        string
	CardTrader         []CardTraderEntry
}

type CardTraderEntry struct {
	ID              int
	MatchType       string
	Expansion       string
	ExpansionCode   string
	CollectorNumber string
	Finishes        []string
	Languages       []string
	Properties      []CardTraderProperty
}

type CardTraderProperty struct {
	Name         string
	Type         string
	DefaultValue any
	Possible     []any
}

type PricingResult struct {
	ProductID        int
	Subtypes         map[string]Price
	Manapool         map[string]float64
	ManapoolQuantity *int
}

type PricingSnapshot struct {
	UpdatedAt string
	Prices    []PricingResult
}

type Price struct {
	Low    *float64
	Market *float64
}

type SKUResult struct {
	ProductID int
	SKUs      []SKU
}

type SKUSnapshot struct {
	UpdatedAt string
	Products  []SKUResult
}

type SKU struct {
	ID        int
	Condition string
	Variant   string
	Language  string
	Market    *float64
	Low       *float64
	High      *float64
	Count     *int
}

type PaginatedProducts struct {
	Items      []Product
	Pagination struct {
		Total    int
		Returned int
		Offset   int
		HasMore  bool
	}
}

type YearRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type ReleaseYearCount struct {
	Year     int `json:"year"`
	SetCount int `json:"set_count"`
}

type ReleaseCountsSummary struct {
	CategoryScope        string             `json:"category_scope"`
	YearRange            YearRange          `json:"year_range"`
	SupplementalIncluded bool               `json:"supplemental_included"`
	TotalSets            int                `json:"total_sets"`
	CountsByYear         []ReleaseYearCount `json:"counts_by_year"`
}

type SetMetadata struct {
	ID             int    `json:"id"`
	CategoryID     int    `json:"category_id"`
	Name           string `json:"name"`
	Abbreviation   string `json:"abbreviation,omitempty"`
	IsSupplemental bool   `json:"is_supplemental"`
	PublishedOn    string `json:"published_on"`
	ProductCount   int    `json:"product_count"`
	SKUCount       int    `json:"sku_count"`
}

type RarityBreakdown struct {
	Rarity             string   `json:"rarity"`
	ProductCount       int      `json:"product_count"`
	MarketCardCount    int      `json:"market_card_count"`
	TotalMarketValue   float64  `json:"total_market_value"`
	HighestMarketValue *float64 `json:"highest_market_value,omitempty"`
}

type NumberingSummary struct {
	NumberedProducts   int      `json:"numbered_products"`
	UnnumberedProducts int      `json:"unnumbered_products"`
	SampleNumbers      []string `json:"sample_numbers,omitempty"`
	HasSlashNumbers    bool     `json:"has_slash_numbers"`
	HasLetterSuffixes  bool     `json:"has_letter_suffixes"`
}

type TopMarketCard struct {
	ProductID   int     `json:"product_id"`
	Name        string  `json:"name"`
	Number      string  `json:"number,omitempty"`
	Rarity      string  `json:"rarity,omitempty"`
	Subtype     string  `json:"subtype"`
	MarketPrice float64 `json:"market_price"`
}

type HighestValueRarity struct {
	Rarity      string  `json:"rarity"`
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	Number      string  `json:"number,omitempty"`
	Subtype     string  `json:"subtype"`
	MarketPrice float64 `json:"market_price"`
}

type SetInsights struct {
	Set                   SetMetadata         `json:"set"`
	ProductCountTotal     int                 `json:"product_count_total"`
	NumberedCardLikeCount int                 `json:"numbered_card_like_count"`
	NumberingSummary      NumberingSummary    `json:"numbering_summary"`
	RarityBreakdown       []RarityBreakdown   `json:"rarity_breakdown"`
	PricingUpdatedAt      string              `json:"pricing_updated_at"`
	SKUUpdatedAt          string              `json:"sku_updated_at"`
	TopMarketCards        []TopMarketCard     `json:"top_market_cards"`
	HighestValueRarity    *HighestValueRarity `json:"highest_value_rarity,omitempty"`
	MarketSumEstimate     float64             `json:"market_sum_estimate"`
	HeuristicNotes        []string            `json:"heuristic_notes,omitempty"`
}
