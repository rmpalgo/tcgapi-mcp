package tcgapi

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi/generated"
)

func (c *Client) SetPricing(ctx context.Context, categoryID, setID int, productID *int) ([]domain.PricingResult, error) {
	resp, err := c.raw.GetSetPricingWithResponse(ctx, generated.CategoryId(categoryID), generated.SetId(setID))
	if err != nil {
		return nil, fmt.Errorf("get set pricing: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus("get set pricing", resp.StatusCode(), resp.Body)
	}

	ids := make([]string, 0, len(resp.JSON200.Prices))
	for id := range resp.JSON200.Prices {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]domain.PricingResult, 0, len(ids))
	for _, id := range ids {
		parsedID, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		if productID != nil && parsedID != *productID {
			continue
		}

		priceInfo := resp.JSON200.Prices[id]
		subtypes := make(map[string]domain.Price)
		if priceInfo.Tcg != nil {
			subtypes = make(map[string]domain.Price, len(*priceInfo.Tcg))
		}
		for subtype, price := range mapOrEmpty(priceInfo.Tcg) {
			subtypes[subtype] = domain.Price{
				Low:    price.Low,
				Market: price.Market,
			}
		}

		manapool := make(map[string]float64)
		if priceInfo.Manapool != nil {
			manapool = make(map[string]float64, len(*priceInfo.Manapool))
		}
		for finish, value := range mapOrEmpty(priceInfo.Manapool) {
			manapool[finish] = value
		}

		out = append(out, domain.PricingResult{
			ProductID:        parsedID,
			Subtypes:         subtypes,
			Manapool:         manapool,
			ManapoolQuantity: clonePointer(priceInfo.MpQty),
		})
	}

	return out, nil
}
