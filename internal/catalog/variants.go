package catalog

type VariantMetadata struct {
	PricingSubtypes []string
	SKUVariants     map[string]string
}

func VariantMetadataByCategory() map[int]VariantMetadata {
	return map[int]VariantMetadata{
		1: {
			PricingSubtypes: []string{"Normal", "Foil"},
			SKUVariants: map[string]string{
				"N": "Normal",
				"F": "Foil",
			},
		},
		2: {
			PricingSubtypes: []string{"Normal", "1st Edition"},
			SKUVariants: map[string]string{
				"N":  "Normal",
				"1E": "1st Edition",
			},
		},
		3: {
			PricingSubtypes: []string{"Normal", "Holofoil", "Reverse Holofoil"},
			SKUVariants: map[string]string{
				"N": "Normal",
				"H": "Holofoil",
				"R": "Reverse Holofoil",
			},
		},
		62: {
			PricingSubtypes: []string{"Normal"},
			SKUVariants: map[string]string{
				"N":  "Normal",
				"CF": "Cold Foil",
				"RF": "Rainbow Foil",
			},
		},
		68: {
			PricingSubtypes: []string{"Normal", "Foil"},
			SKUVariants: map[string]string{
				"N": "Normal",
				"F": "Foil",
			},
		},
		71: {
			PricingSubtypes: []string{"Normal", "Cold Foil", "Holofoil"},
			SKUVariants: map[string]string{
				"N":  "Normal",
				"CF": "Cold Foil",
				"H":  "Holofoil",
			},
		},
	}
}
