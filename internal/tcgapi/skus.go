package tcgapi

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi/generated"
)

func (c *Client) SetSKUs(ctx context.Context, categoryID, setID int, productID *int) ([]domain.SKUResult, error) {
	resp, err := c.raw.GetSetSkusWithResponse(ctx, generated.CategoryId(categoryID), generated.SetId(setID))
	if err != nil {
		return nil, fmt.Errorf("get set skus: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus("get set skus", resp.StatusCode(), resp.Body)
	}

	productIDs := make([]string, 0, len(resp.JSON200.Products))
	for id := range resp.JSON200.Products {
		productIDs = append(productIDs, id)
	}
	sort.Strings(productIDs)

	out := make([]domain.SKUResult, 0, len(productIDs))
	for _, id := range productIDs {
		parsedID, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		if productID != nil && parsedID != *productID {
			continue
		}

		skuIDs := make([]string, 0, len(resp.JSON200.Products[id]))
		for skuID := range resp.JSON200.Products[id] {
			skuIDs = append(skuIDs, skuID)
		}
		sort.Strings(skuIDs)

		skus := make([]domain.SKU, 0, len(skuIDs))
		for _, skuID := range skuIDs {
			raw := resp.JSON200.Products[id][skuID]
			parsedSKU, err := strconv.Atoi(skuID)
			if err != nil {
				continue
			}

			skus = append(skus, domain.SKU{
				ID:        parsedSKU,
				Condition: string(raw.Cnd),
				Variant:   raw.Var,
				Language:  raw.Lng,
				Market:    raw.Mkt,
				Low:       raw.Low,
				High:      raw.Hi,
				Count:     raw.Cnt,
			})
		}

		out = append(out, domain.SKUResult{
			ProductID: parsedID,
			SKUs:      skus,
		})
	}

	return out, nil
}
