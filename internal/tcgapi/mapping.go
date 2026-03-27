package tcgapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/rmpalgo/tcgapi-mcp/internal/domain"
	"github.com/rmpalgo/tcgapi-mcp/internal/tcgapi/generated"
)

func unexpectedStatus(operation string, statusCode int, body []byte) error {
	body = bodyPreview(body, 4096)
	if len(body) == 0 {
		return fmt.Errorf("%s: unexpected status %d", operation, statusCode)
	}
	return fmt.Errorf("%s: unexpected status %d body=%q", operation, statusCode, string(body))
}

func bodyPreview(body []byte, max int) []byte {
	body = bytesTrimSpace(body)
	if len(body) <= max {
		return body
	}
	return body[:max]
}

func bytesTrimSpace(body []byte) []byte {
	return []byte(strings.TrimSpace(string(body)))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatDate(date openapi_types.Date) string {
	return date.String()
}

func formatDatePtr(date *openapi_types.Date) string {
	if date == nil {
		return ""
	}
	return date.String()
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func float32PointerTo64(v *float32) *float64 {
	if v == nil {
		return nil
	}
	value := float64(*v)
	return &value
}

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneSlicePtr[T any](value *[]T) []T {
	if value == nil {
		return nil
	}
	return append([]T(nil), (*value)...)
}

func mapOrEmpty[V any](value *map[string]V) map[string]V {
	if value == nil {
		return map[string]V{}
	}
	return *value
}

func mapColors(colors *[]generated.ProductColors) []string {
	if colors == nil {
		return nil
	}

	out := make([]string, 0, len(*colors))
	for _, color := range *colors {
		out = append(out, string(color))
	}
	return out
}

func mapCardTraderEntries(entries *[]generated.CardTraderEntry) []domain.CardTraderEntry {
	if entries == nil {
		return nil
	}

	out := make([]domain.CardTraderEntry, 0, len(*entries))
	for _, entry := range *entries {
		out = append(out, domain.CardTraderEntry{
			ID:              intValue(entry.Id),
			MatchType:       matchTypeValue(entry.MatchType),
			Expansion:       stringValue(entry.Expansion),
			ExpansionCode:   stringValue(entry.ExpansionCode),
			CollectorNumber: stringValue(entry.CollectorNumber),
			Finishes:        cloneSlicePtr(entry.Finishes),
			Languages:       cloneSlicePtr(entry.Languages),
			Properties:      mapCardTraderProperties(entry.Properties),
		})
	}

	return out
}

func mapCardTraderProperties(properties *[]generated.CardTraderProperty) []domain.CardTraderProperty {
	if properties == nil {
		return nil
	}

	out := make([]domain.CardTraderProperty, 0, len(*properties))
	for _, property := range *properties {
		out = append(out, domain.CardTraderProperty{
			Name:         property.Name,
			Type:         propertyTypeValue(property.Type),
			DefaultValue: jsonValue(property.DefaultValue),
			Possible:     mapPossibleValues(property.PossibleValues),
		})
	}

	return out
}

func mapPossibleValues(values *[]generated.CardTraderProperty_PossibleValues_Item) []any {
	if values == nil {
		return nil
	}

	out := make([]any, 0, len(*values))
	for _, value := range *values {
		out = append(out, jsonValue(value))
	}
	return out
}

func jsonValue(value any) any {
	if value == nil {
		return nil
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil
	}

	return decoded
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func matchTypeValue(value *generated.CardTraderEntryMatchType) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func propertyTypeValue(value *generated.CardTraderPropertyType) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
