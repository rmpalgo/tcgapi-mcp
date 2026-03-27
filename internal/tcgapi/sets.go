package tcgapi

import (
	"context"
	"fmt"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi/generated"
)

func (c *Client) CategorySets(ctx context.Context, categoryID int) ([]domain.SetSummary, error) {
	resp, err := c.raw.GetCategorySetsWithResponse(ctx, generated.CategoryId(categoryID))
	if err != nil {
		return nil, fmt.Errorf("get category sets: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus("get category sets", resp.StatusCode(), resp.Body)
	}

	return mapSets(resp.JSON200.CategoryId, resp.JSON200.Sets), nil
}

func (c *Client) SearchSets(ctx context.Context, categoryID int, query string) ([]domain.SetSummary, error) {
	resp, err := c.raw.SearchSetsWithResponse(ctx, generated.CategoryId(categoryID), &generated.SearchSetsParams{
		Q: query,
	})
	if err != nil {
		return nil, fmt.Errorf("search sets: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus("search sets", resp.StatusCode(), resp.Body)
	}

	return mapSets(resp.JSON200.CategoryId, resp.JSON200.Sets), nil
}

func mapSets(categoryID int, sets []generated.Set) []domain.SetSummary {
	out := make([]domain.SetSummary, 0, len(sets))
	for _, set := range sets {
		out = append(out, domain.SetSummary{
			ID:             set.Id,
			CategoryID:     categoryID,
			Name:           set.Name,
			Abbreviation:   stringValue(set.Abbreviation),
			IsSupplemental: set.IsSupplemental,
			PublishedOn:    formatDate(set.PublishedOn),
			ProductCount:   set.ProductCount,
			SKUCount:       set.SkuCount,
			APIURL:         set.ApiUrl,
			PricingURL:     set.PricingUrl,
			SKUsURL:        set.SkusUrl,
		})
	}
	return out
}
