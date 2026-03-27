package main

import (
	"context"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	specPath := "openapi/tcgtracking-openapi.yaml"
	if len(os.Args) > 2 {
		return fmt.Errorf("usage: openapi-validate [spec-path]")
	}
	if len(os.Args) == 2 {
		specPath = os.Args[1]
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return fmt.Errorf("load OpenAPI spec: %w", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return fmt.Errorf("validate OpenAPI spec: %w", err)
	}

	fmt.Printf(
		"openapi ok: version=%s paths=%d\n",
		doc.OpenAPI,
		len(doc.Paths.Map()),
	)
	return nil
}
