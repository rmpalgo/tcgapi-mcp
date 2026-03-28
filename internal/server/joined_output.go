package server

import (
	"context"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

func (s *Server) pricingOutput(ctx context.Context, categoryID, setID int, productID *int) (getSetPricingOutput, error) {
	pricing, err := s.api.SetPricing(ctx, categoryID, setID, productID)
	if err != nil {
		return getSetPricingOutput{}, err
	}

	products, err := s.api.SetProducts(ctx, categoryID, setID)
	if err != nil {
		return getSetPricingOutput{}, err
	}

	return getSetPricingOutput{
		CategoryID: categoryID,
		SetID:      setID,
		ProductID:  productID,
		UpdatedAt:  pricing.UpdatedAt,
		Prices:     joinPricingResults(pricing.Prices, products),
	}, nil
}

func (s *Server) skuOutput(ctx context.Context, categoryID, setID int, productID *int) (getSetSKUsOutput, error) {
	skus, err := s.api.SetSKUs(ctx, categoryID, setID, productID)
	if err != nil {
		return getSetSKUsOutput{}, err
	}

	products, err := s.api.SetProducts(ctx, categoryID, setID)
	if err != nil {
		return getSetSKUsOutput{}, err
	}

	return getSetSKUsOutput{
		CategoryID: categoryID,
		SetID:      setID,
		ProductID:  productID,
		UpdatedAt:  skus.UpdatedAt,
		Products:   joinSKUResults(skus.Products, products),
	}, nil
}

func joinPricingResults(results []domain.PricingResult, products []domain.Product) []domain.PricingResult {
	summaries := productSummaryIndex(products)
	joined := make([]domain.PricingResult, 0, len(results))
	for _, result := range results {
		current := result
		current.Product = summaries[result.ProductID]
		joined = append(joined, current)
	}
	return joined
}

func joinSKUResults(results []domain.SKUResult, products []domain.Product) []domain.SKUResult {
	summaries := productSummaryIndex(products)
	joined := make([]domain.SKUResult, 0, len(results))
	for _, result := range results {
		current := result
		current.Product = summaries[result.ProductID]
		joined = append(joined, current)
	}
	return joined
}

func productSummaryIndex(products []domain.Product) map[int]*domain.ProductSummary {
	index := make(map[int]*domain.ProductSummary, len(products))
	for _, product := range products {
		summary := &domain.ProductSummary{
			ID:     product.ID,
			Name:   product.Name,
			Number: product.Number,
			Rarity: product.Rarity,
		}
		index[product.ID] = summary
	}
	return index
}
