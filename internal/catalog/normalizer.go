package catalog

import "github.com/rmpalgo/tcgapi-mcp/internal/domain"

type NormalizedProduct struct {
	ID                 int                      `json:"id"`
	Name               string                   `json:"name"`
	SetName            string                   `json:"set_name,omitempty"`
	Number             string                   `json:"number,omitempty"`
	Rarity             string                   `json:"rarity,omitempty"`
	ImageURL           string                   `json:"image_url,omitempty"`
	ScryfallID         string                   `json:"scryfall_id,omitempty"`
	MtgjsonUUID        string                   `json:"mtgjson_uuid,omitempty"`
	Colors             []string                 `json:"colors,omitempty"`
	ManaValue          *float64                 `json:"mana_value,omitempty"`
	Finishes           []string                 `json:"finishes,omitempty"`
	IsPresale          bool                     `json:"is_presale,omitempty"`
	PresaleReleaseDate string                   `json:"presale_release_date,omitempty"`
	CardTrader         []domain.CardTraderEntry `json:"cardtrader,omitempty"`
}

func NormalizeProduct(product domain.Product) NormalizedProduct {
	return NormalizedProduct{
		ID:                 product.ID,
		Name:               product.Name,
		SetName:            product.SetName,
		Number:             product.Number,
		Rarity:             product.Rarity,
		ImageURL:           product.ImageURL,
		ScryfallID:         product.ScryfallID,
		MtgjsonUUID:        product.MtgjsonUUID,
		Colors:             append([]string(nil), product.Colors...),
		ManaValue:          product.ManaValue,
		Finishes:           append([]string(nil), product.Finishes...),
		IsPresale:          product.IsPresale,
		PresaleReleaseDate: product.PresaleReleaseDate,
		CardTrader:         append([]domain.CardTraderEntry(nil), product.CardTrader...),
	}
}

func PaginateProducts(products []domain.Product, limit, offset int) domain.PaginatedProducts {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(products) {
		offset = len(products)
	}

	end := offset + limit
	if end > len(products) {
		end = len(products)
	}

	items := append([]domain.Product(nil), products[offset:end]...)

	var paginated domain.PaginatedProducts
	paginated.Items = items
	paginated.Pagination.Total = len(products)
	paginated.Pagination.Returned = len(items)
	paginated.Pagination.Offset = offset
	paginated.Pagination.HasMore = end < len(products)
	return paginated
}
