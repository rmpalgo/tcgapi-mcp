package catalog

import (
	"testing"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
)

func TestResolverResolvesAliasesAndNames(t *testing.T) {
	t.Parallel()

	r := NewResolver([]domain.Category{
		{ID: 1, Name: "Magic", DisplayName: "Magic: The Gathering"},
		{ID: 3, Name: "Pokemon", DisplayName: "Pokemon"},
	}, DefaultAliases())

	testCases := []struct {
		name  string
		input string
		want  int
	}{
		{name: "alias", input: "mtg", want: 1},
		{name: "display name", input: "Magic: The Gathering", want: 1},
		{name: "numeric", input: "3", want: 3},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := r.ResolveCategory(tc.input)
			if err != nil {
				t.Fatalf("ResolveCategory(%q) error = %v", tc.input, err)
			}
			if got.ID != tc.want {
				t.Fatalf("ResolveCategory(%q) id = %d, want %d", tc.input, got.ID, tc.want)
			}
		})
	}
}

func TestResolverUnknownCategory(t *testing.T) {
	t.Parallel()

	r := NewResolver([]domain.Category{{ID: 1, Name: "Magic", DisplayName: "Magic: The Gathering"}}, DefaultAliases())
	if _, err := r.ResolveCategory("unknown"); err == nil {
		t.Fatal("expected error for unknown category")
	}
}
