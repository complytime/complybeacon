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

# self signed cert related
CERT_DIR := hack/self-signed-cert
OPENSSL_CNF := $(CERT_DIR)/openssl.cnf

# The default target. Running 'make' with no arguments will execute this.
all: test build

# ------------------------------------------------------------------------------
# Test Target
# ------------------------------------------------------------------------------
test: ## Runs unit tests with coverage for every module in the monorepo.
	@for m in $(MODULES); do \
		echo "========================================================================================================="; \
		echo "Running tests for $$m..."; \
		echo "========================================================================================================="; \
		(cd $$m && go test -v -coverprofile=coverage.out -covermode=atomic ./...); \
		if [ $$? -ne 0 ]; then \
			echo "Tests failed for module: $$m"; \
			exit 1; \
		fi; \
		echo "Coverage summary for $$m:"; \
		(cd $$m && go tool cover -func=coverage.out | tail -n1) || true; \
		echo "-------------------"; \
	done
	@echo "--- All tests passed! ---"
.PHONY: test

test-race: ## Runs tests with race detection
	@for m in $(MODULES); do \
		echo "Running tests with race detection for $$m..."; \
		(cd $$m && go test -v -race ./...); \
		if [ $$? -ne 0 ]; then \
			echo "Tests failed for module: $$m"; \
			exit 1; \
		fi; \
	done
	@echo "--- All tests passed with race detection! ---"
.PHONY: test-race

# ------------------------------------------------------------------------------
# Dependencies for all modules
# ------------------------------------------------------------------------------
deps: ## Tidy, verify, download, and vendor deps for all modules
	@for m in $(MODULES); do \
		echo "Processing deps for $$m..."; \
		(cd $$m && go mod tidy && go mod verify && go mod download && go mod vendor); \
		if [ $$? -ne 0 ]; then \
			echo "Deps failed for module: $$m"; \
			exit 1; \
		fi; \
		echo "-------------------"; \
	done
	@echo "--- Deps completed for all modules ---"
.PHONY: deps

coverage-report: test ## Generate HTML coverage report and show summary
	@for m in $(MODULES); do \
		echo "Generating coverage report for $$m..."; \
		(cd $$m && go tool cover -html=coverage.out -o coverage.html); \
		echo "Coverage summary for $$m:"; \
		(cd $$m && go tool cover -func=coverage.out | tail -n1) || true; \
		echo "-------------------"; \
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

generate-self-signed-cert: ## Generate self-signed certificates for compass and truthbeam
	@echo "--- Generating self-signed certificates in $(CERT_DIR) ---"
	# 1. Create the new Root CA key
	@openssl genrsa -out $(CERT_DIR)/truthbeam.key 2048
	# 2. Create the new Root CA certificate
	@openssl req -x509 -new -nodes -key $(CERT_DIR)/truthbeam.key -sha256 -days 365 \
		-subj "/CN=ComplyBeacon Root CA" \
		-extensions v3_ca -config $(OPENSSL_CNF) \
		-out $(CERT_DIR)/truthbeam.crt
	# 3. Create the server's private key
	@openssl genrsa -out $(CERT_DIR)/compass.key 2048
	@chmod a+r $(CERT_DIR)/compass.key
	# 4. Create a Certificate Signing Request (CSR) for the server
	@openssl req -new -key $(CERT_DIR)/compass.key -out $(CERT_DIR)/compass.csr -config $(OPENSSL_CNF)
	# 5. Use your new Root CA to sign the server's CSR
	@openssl x509 -req -in $(CERT_DIR)/compass.csr -CA $(CERT_DIR)/truthbeam.crt -CAkey $(CERT_DIR)/truthbeam.key -CAcreateserial \
		-out $(CERT_DIR)/compass.crt -days 365 -sha256 \
		-extfile $(OPENSSL_CNF) -extensions v3_req
	@echo "--- Certificates generated successfully ---"
.PHONY: generate-self-signed-cert

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
	weaver registry generate -r model --templates templates go --param package_name="applier" truthbeam/internal/applier
.PHONY: weaver-codegen

weaver-check: ## Model schema check
	weaver registry check -r model
.PHONY: weaver-check

weaver-semantic-check: ## Validate logs against semantic conventions
	@echo "Generating test OCSF and Gemara logs..."
	cd proofwatch && go run -mod=readonly ./cmd/validate-logs both /tmp/test-enriched-logs.json
	@echo "Validating with weaver live-check (development stability warnings suppressed)..."
	@cat /tmp/test-enriched-logs.json | \
		weaver registry live-check -r model --input-source stdin --input-format json 2>&1 | \
		(grep -v "development.*Is not stable" || true)
	@echo ""
	@echo "---------------------------------------------------------------"
	@echo "Note: Development stability warnings have been suppressed."
	@echo "To see all advisories including development stability warnings, run:"
	@echo "  make weaver-semantic-check-verbose"
.PHONY: weaver-semantic-check

weaver-semantic-check-verbose: ## Validate with verbose output
	@echo "Generating test OCSF and Gemara logs..."
	cd proofwatch && go run -mod=readonly ./cmd/validate-logs both /tmp/test-enriched-logs.json
	@echo "Validating with weaver live-check (showing all advisories)..."
	@cat /tmp/test-enriched-logs.json | \
		weaver registry live-check -r model --input-source stdin --input-format json
.PHONY: weaver-semantic-check-verbose
# ------------------------------------------------------------------------------
# Help Target
# Prints a friendly help message.
# ------------------------------------------------------------------------------
help: ## Display this help screen
	@grep -E '^[a-z.A-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help
