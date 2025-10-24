# ==============================================================================
# Monorepo Makefile
# Assisted by: Gemini 2.5 Pro
# ==============================================================================
# This Makefile automates common tasks for a Go monorepo with multiple modules.
# It assumes a structure where each application is a module with its main
# package located in a 'cmd/' subdirectory.
#
# Usage:
#   make all         - Runs tests and then builds all binaries
#   make test        - Runs tests for all modules
#   make build       - Builds all binaries and places them in the ./bin directory
#   make clean       - Removes generated binaries and build artifacts
#   make help        - Displays this help message
# ==============================================================================

# Define a list of your Go modules.
# Add or remove modules here as your project evolves.
# The path should be relative to the Makefile's location.
MODULES := ./compass ./proofwatch ./truthbeam
BUILD := ./compass

# The directory where the compiled binaries will be placed.
BIN_DIR := bin

# The default target. Running 'make' with no arguments will execute this.
all: test build

# ------------------------------------------------------------------------------
# Test Target
# ------------------------------------------------------------------------------
test: ## Runs unit tests for every module in the monorepo.
	@for m in $(MODULES); do \
		(cd $$m && go test -v ./...); \
		if [ $$? -ne 0 ]; then \
			echo "Tests failed for module: $$m"; \
			exit 1; \
		fi; \
	done
	@echo "--- All tests passed! ---"
.PHONY: test

test-coverage: ## Runs tests with coverage for all modules
	@for m in $(MODULES); do \
		echo "Running tests with coverage for $$m..."; \
		(cd $$m && go test -v -race -coverprofile=coverage.out -covermode=atomic ./...); \
		if [ $$? -ne 0 ]; then \
			echo "Tests failed for module: $$m"; \
			exit 1; \
		fi; \
	done
	@echo "--- All tests passed with coverage! ---"
.PHONY: test-coverage

coverage-report: test-coverage ## Generate HTML coverage report
	@for m in $(MODULES); do \
		echo "Generating coverage report for $$m..."; \
		(cd $$m && go tool cover -html=coverage.out -o coverage.html); \
	done
	@echo "--- Coverage reports generated! ---"
.PHONY: coverage-report

# ------------------------------------------------------------------------------
# Build Target
# This assumes the main package is in a subdirectory named 'cmd/'.
# ------------------------------------------------------------------------------
build: ## Builds a binary for each module and places it in the $(BIN_DIR) directory.
	@mkdir -p $(BIN_DIR)
	@for m in $(BUILD); do \
    		(cd $$m && go build -v -o ../$(BIN_DIR)/ ./cmd/... ); \
    		if [ $$? -ne 0 ]; then \
    			echo "Build failed for module: $$m"; \
    			exit 1; \
    		fi; \
    done
	@echo "--- All binaries built successfully ---"
.PHONY: build


clean: ## Removes all generated binaries and Go build caches.
	@echo "--- Cleaning up build artifacts ---"
	@rm -rf $(BIN_DIR)
	@go clean -modcache
	@echo "--- Cleanup complete ---"
.PHONY: clean

workspace: # Setup a go workspace with all modules
		@go work init && go work use $(MODULES)
.PHONY: workspace

#------------------------------------------------------------------------------
# Demo
#------------------------------------------------------------------------------

deploy: ## Deploy infra
	podman-compose -f compose.yaml up
.PHONY: deploy

undeploy: ## Undeploy container stack
	podman-compose -f compose.yaml down -v
.PHONY: undeploy

#------------------------------------------------------------------------------
# Generate
#------------------------------------------------------------------------------

api-codegen: ## Runs go generate for all the modules
	@for m in $(MODULES); do \
		(cd $$m && go generate ./...); \
		if [ $$? -ne 0 ]; then \
			echo "Codegen failed for module: $$m"; \
			exit 1; \
		fi; \
	done
.PHONY: api-codegen

#------------------------------------------------------------------------------
# Weaver - See documenation for more information https://github.com/open-telemetry/weaver?tab=readme-ov-file
#------------------------------------------------------------------------------

weaver-docsgen: ## Generate docs
	weaver registry generate -r model --templates "https://github.com/open-telemetry/semantic-conventions/archive/refs/tags/v1.34.0.zip[templates]" markdown docs
.PHONY: weaver-docsgen

weaver-codegen: ## Generate Go code
	weaver registry generate -r model --templates templates go --param package_name="proofwatch" proofwatch
	weaver registry generate -r model --templates templates go --param package_name="client" truthbeam/internal/client
.PHONY: weaver-codegen

weaver-check: ## Model schema check
	weaver registry check -r model
# ------------------------------------------------------------------------------
# Help Target
# Prints a friendly help message.
# ------------------------------------------------------------------------------
help: ## Display this help screen
	@grep -E '^[a-z.A-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help
