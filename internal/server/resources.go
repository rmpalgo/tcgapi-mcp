package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	s.resources = []*mcp.Resource{
		{
			Name:        "meta",
			URI:         "tcg:///meta",
			Description: "API status, counts, and freshness metadata.",
			MIMEType:    "application/json",
		},
		{
			Name:        "categories",
			URI:         "tcg:///categories",
			Description: "Full category list.",
			MIMEType:    "application/json",
		},
	}

	s.resourceTemplates = []*mcp.ResourceTemplate{
		{
			Name:        "category-sets",
			URITemplate: "tcg:///{categoryId}/sets",
			Description: "All sets for a game category.",
			MIMEType:    "application/json",
		},
		{
			Name:        "set-products",
			URITemplate: "tcg:///{categoryId}/sets/{setId}",
			Description: "Products in a set.",
			MIMEType:    "application/json",
		},
		{
			Name:        "set-pricing",
			URITemplate: "tcg:///{categoryId}/sets/{setId}/pricing",
			Description: "Pricing data for a set.",
			MIMEType:    "application/json",
		},
		{
			Name:        "set-skus",
			URITemplate: "tcg:///{categoryId}/sets/{setId}/skus",
			Description: "SKU detail for a set.",
			MIMEType:    "application/json",
		},
	}

	for _, resource := range s.resources {
		s.raw.AddResource(resource, s.readResource)
	}
	for _, template := range s.resourceTemplates {
		s.raw.AddResourceTemplate(template, s.readResource)
	}
}

func (s *Server) readResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	value, err := s.readResourceValue(ctx, req.Params.URI)
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal resource %q: %w", req.Params.URI, err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *Server) readResourceValue(ctx context.Context, rawURI string) (any, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil || parsed.Scheme != "tcg" {
		return nil, mcp.ResourceNotFoundError(rawURI)
	}

	parts := splitResourcePath(parsed.Path)
	switch {
	case len(parts) == 1 && parts[0] == "meta":
		return s.api.Meta(ctx)
	case len(parts) == 1 && parts[0] == "categories":
		categories := s.resolver.Categories()
		if len(categories) == 0 {
			categories, err = s.api.Categories(ctx)
			if err != nil {
				return nil, err
			}
		}
		return listCategoriesOutput{Categories: categories}, nil
	case len(parts) == 2 && parts[1] == "sets":
		categoryID, err := parseResourceInt(parts[0], rawURI)
		if err != nil {
			return nil, err
		}
		sets, err := s.api.CategorySets(ctx, categoryID)
		if err != nil {
			return nil, err
		}
		return categorySetsOutput{
			CategoryID: categoryID,
			Sets:       sets,
		}, nil
	case len(parts) == 3 && parts[1] == "sets":
		categoryID, setID, err := parseCategorySetParts(parts[0], parts[2], rawURI)
		if err != nil {
			return nil, err
		}
		products, err := s.api.SetProducts(ctx, categoryID, setID)
		if err != nil {
			return nil, err
		}
		return fullProductsOutput(categoryID, setID, products), nil
	case len(parts) == 4 && parts[1] == "sets" && parts[3] == "pricing":
		categoryID, setID, err := parseCategorySetParts(parts[0], parts[2], rawURI)
		if err != nil {
			return nil, err
		}
		pricing, err := s.api.SetPricing(ctx, categoryID, setID, nil)
		if err != nil {
			return nil, err
		}
		return getSetPricingOutput{
			CategoryID: categoryID,
			SetID:      setID,
			UpdatedAt:  pricing.UpdatedAt,
			Prices:     pricing.Prices,
		}, nil
	case len(parts) == 4 && parts[1] == "sets" && parts[3] == "skus":
		categoryID, setID, err := parseCategorySetParts(parts[0], parts[2], rawURI)
		if err != nil {
			return nil, err
		}
		skus, err := s.api.SetSKUs(ctx, categoryID, setID, nil)
		if err != nil {
			return nil, err
		}
		return getSetSKUsOutput{
			CategoryID: categoryID,
			SetID:      setID,
			UpdatedAt:  skus.UpdatedAt,
			Products:   skus.Products,
		}, nil
	default:
		return nil, mcp.ResourceNotFoundError(rawURI)
	}
}

func splitResourcePath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func parseCategorySetParts(categoryPart, setPart, rawURI string) (int, int, error) {
	categoryID, err := parseResourceInt(categoryPart, rawURI)
	if err != nil {
		return 0, 0, err
	}

	setID, err := parseResourceInt(setPart, rawURI)
	if err != nil {
		return 0, 0, err
	}

	return categoryID, setID, nil
}

func parseResourceInt(part, rawURI string) (int, error) {
	value, err := strconv.Atoi(part)
	if err != nil {
		return 0, mcp.ResourceNotFoundError(rawURI)
	}
	return value, nil
}
