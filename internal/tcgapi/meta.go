package tcgapi

import (
	"context"
	"fmt"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

func (c *Client) Meta(ctx context.Context) (domain.Meta, error) {
	resp, err := c.raw.GetMetaWithResponse(ctx)
	if err != nil {
		return domain.Meta{}, fmt.Errorf("get meta: %w", err)
	}
	if resp.JSON200 == nil {
		return domain.Meta{}, unexpectedStatus("get meta", resp.StatusCode(), resp.Body)
	}

	return domain.Meta{
		LastUpdated:     formatTime(resp.JSON200.LastUpdated),
		PricingUpdated:  formatTime(resp.JSON200.PricingUpdated),
		TotalCategories: resp.JSON200.TotalCategories,
		TotalSets:       resp.JSON200.TotalSets,
		TotalProducts:   resp.JSON200.TotalProducts,
		Version:         resp.JSON200.Version,
		Documentation:   resp.JSON200.Documentation,
	}, nil
}
