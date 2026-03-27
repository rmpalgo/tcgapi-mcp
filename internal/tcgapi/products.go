package tcgapi

import (
	"context"
	"fmt"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi/generated"
)

func (c *Client) SetProducts(ctx context.Context, categoryID, setID int) ([]domain.Product, error) {
	resp, err := c.raw.GetSetProductsWithResponse(ctx, generated.CategoryId(categoryID), generated.SetId(setID))
	if err != nil {
		return nil, fmt.Errorf("get set products: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus("get set products", resp.StatusCode(), resp.Body)
	}

	out := make([]domain.Product, 0, len(resp.JSON200.Products))
	for _, product := range resp.JSON200.Products {
		out = append(out, mapProduct(resp.JSON200.SetId, product))
	}

	return out, nil
}

func mapProduct(setID int, product generated.Product) domain.Product {
	return domain.Product{
		ID:                 product.Id,
		SetID:              setID,
		Name:               product.Name,
		CleanName:          product.CleanName,
		SetName:            stringValue(product.SetName),
		SetAbbreviation:    stringValue(product.SetAbbr),
		Number:             stringValue(product.Number),
		Rarity:             stringValue(product.Rarity),
		ImageURL:           product.ImageUrl,
		TCGPlayerURL:       product.TcgplayerUrl,
		ManapoolURL:        stringValue(product.ManapoolUrl),
		ScryfallID:         stringValue(product.ScryfallId),
		MtgjsonUUID:        stringValue(product.MtgjsonUuid),
		CardMarketID:       clonePointer(product.CardmarketId),
		CardTraderID:       clonePointer(product.CardtraderId),
		Colors:             mapColors(product.Colors),
		ManaValue:          float32PointerTo64(product.ManaValue),
		Finishes:           cloneSlicePtr(product.Finishes),
		IsPresale:          boolValue(product.IsPresale),
		PresaleReleaseDate: formatDatePtr(product.PresaleReleaseDate),
		PresaleNote:        stringValue(product.PresaleNote),
		CardTrader:         mapCardTraderEntries(product.Cardtrader),
	}
}
