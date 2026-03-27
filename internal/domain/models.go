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
