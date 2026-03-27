APP := tcgapi-mcp
BIN_DIR := bin
OPENAPI_SPEC := openapi/tcgtracking-openapi.yaml
OPENAPI_CLIENT_CONFIG := openapi/oapi-codegen.client.yaml
OPENAPI_TYPES_CONFIG := openapi/oapi-codegen.types.yaml
GO_CACHE_DIR ?= /tmp/tcgapi-mcp-go-cache
GO := GOCACHE=$(GO_CACHE_DIR) go

.PHONY: help build test fmt fmt-check lint clean run release-snapshot generate generate-check openapi-validate

help:
	@printf "Targets:\n"
	@printf "  make build    Build ./$(BIN_DIR)/$(APP)\n"
	@printf "  make test     Run go test ./...\n"
	@printf "  make fmt      Run gofmt over the repo\n"
	@printf "  make fmt-check Fail if any Go files are not gofmt formatted\n"
	@printf "  make lint     Run go vet ./...\n"
	@printf "  make clean    Remove build artifacts\n"
	@printf "  make run      Run the MCP server\n"
	@printf "  make release-snapshot Run a local Goreleaser snapshot build\n"
	@printf "  make openapi-validate Validate $(OPENAPI_SPEC)\n"
	@printf "  make generate Regenerate OpenAPI client and types\n"
	@printf "  make generate-check Fail if generated OpenAPI artifacts are stale\n"
	@printf "  make ...      Uses GOCACHE=$(GO_CACHE_DIR) by default (override with GO_CACHE_DIR=/path)\n"

build:
	$(GO) build -o ./$(BIN_DIR)/$(APP) ./cmd/$(APP)

test:
	$(GO) test ./...

fmt:
	gofmt -w ./cmd ./internal ./tools

fmt-check:
	@files="$$(find ./cmd ./internal ./tools -name '*.go' -type f -print)"; \
	if [ -z "$$files" ]; then \
		exit 0; \
	fi; \
	unformatted="$$(gofmt -l $$files)"; \
	if [ -n "$$unformatted" ]; then \
		printf 'The following files need gofmt:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

lint:
	$(GO) vet ./...

clean:
	rm -rf ./$(BIN_DIR)

run:
	$(GO) run ./cmd/$(APP)

release-snapshot:
	./scripts/release-smoke.sh

openapi-validate:
	./scripts/validate-openapi.sh $(OPENAPI_SPEC)

generate:
	./scripts/generate-openapi.sh

generate-check:
	./scripts/check-generated.sh
