package catalog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

type Resolver interface {
	ResolveCategory(input string) (domain.Category, error)
	ResolveCategoryID(input string) (int, error)
	Categories() []domain.Category
}

type resolver struct {
	categories []domain.Category
	nameIndex  map[string]int
	aliasIndex map[string]int
	byID       map[int]domain.Category
}

func NewResolver(categories []domain.Category, aliases map[string]int) Resolver {
	r := &resolver{
		categories: append([]domain.Category(nil), categories...),
		nameIndex:  make(map[string]int, len(categories)*2),
		aliasIndex: make(map[string]int, len(aliases)),
		byID:       make(map[int]domain.Category, len(categories)),
	}

	for _, category := range categories {
		r.byID[category.ID] = category
		if key := normalize(category.Name); key != "" {
			r.nameIndex[key] = category.ID
		}
		if key := normalize(category.DisplayName); key != "" {
			r.nameIndex[key] = category.ID
		}
	}

	for alias, id := range aliases {
		r.aliasIndex[normalize(alias)] = id
	}

	return r
}

func DefaultAliases() map[string]int {
	return map[string]int{
		"magic":               1,
		"mtg":                 1,
		"magic the gathering": 1,
		"yugioh":              2,
		"yu-gi-oh":            2,
		"yu-gi-oh!":           2,
		"ygo":                 2,
		"pokemon":             3,
		"pokémon":             3,
		"ptcg":                3,
		"fab":                 62,
		"flesh and blood":     62,
		"flesh & blood":       62,
		"one piece":           68,
		"optcg":               68,
		"one piece card game": 68,
		"lorcana":             71,
		"disney lorcana":      71,
		"pokemon japan":       85,
		"pokemon jp":          85,
		"digimon":             63,
		"digimon card game":   63,
		"star wars unlimited": 79,
		"swu":                 79,
		"weiss schwarz":       20,
		"weiss":               20,
	}
}

func (r *resolver) ResolveCategory(input string) (domain.Category, error) {
	id, err := r.ResolveCategoryID(input)
	if err != nil {
		return domain.Category{}, err
	}

	category, ok := r.byID[id]
	if !ok {
		return domain.Category{}, fmt.Errorf("category id %d is not loaded", id)
	}

	return category, nil
}

func (r *resolver) ResolveCategoryID(input string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("category input must not be empty")
	}

	if id, err := strconv.Atoi(input); err == nil {
		return id, nil
	}

	key := normalize(input)
	if id, ok := r.aliasIndex[key]; ok {
		return id, nil
	}
	if id, ok := r.nameIndex[key]; ok {
		return id, nil
	}

	return 0, fmt.Errorf("unknown category: %q", input)
}

func (r *resolver) Categories() []domain.Category {
	return append([]domain.Category(nil), r.categories...)
}

func normalize(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}
