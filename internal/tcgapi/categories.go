package tcgapi

import (
	"context"
	"fmt"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

func (c *Client) Categories(ctx context.Context) ([]domain.Category, error) {
	resp, err := c.raw.GetCategoriesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus("get categories", resp.StatusCode(), resp.Body)
	}

	out := make([]domain.Category, 0, len(resp.JSON200.Categories))
	for _, category := range resp.JSON200.Categories {
		out = append(out, domain.Category{
			ID:           category.Id,
			Name:         category.Name,
			DisplayName:  category.DisplayName,
			ProductCount: category.ProductCount,
			SetCount:     category.SetCount,
			APIURL:       category.ApiUrl,
		})
	}

	return out, nil
}
