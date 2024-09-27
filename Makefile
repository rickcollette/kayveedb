# Variables
APP_NAME := kayveedb
VERSION_FILE := VERSION
VERSION_GO := lib/kayveedb.go
README_FILE := README.md

# Read the current version from the VERSION file
CURRENT_VERSION := $(shell cat $(VERSION_FILE))
VERSION_MAJOR := $(word 1, $(subst ., ,$(CURRENT_VERSION)))
VERSION_MINOR := $(word 2, $(subst ., ,$(CURRENT_VERSION)))
VERSION_PATCH := $(word 3, $(subst ., ,$(CURRENT_VERSION)))

# Default target
all: build

# Build the package (run go mod tidy to ensure module files are up to date)
build:
	@echo "Tidying up Go modules for $(APP_NAME)..."
	@go mod tidy
	@echo "$(APP_NAME) package is ready."

# Run tests
test:
	@echo "Running tests for $(APP_NAME)..."
	@go test ./...
	@echo "Tests completed."

# Clean up (although nothing to clean for a package, this can remove the module cache)
clean:
	@echo "Cleaning Go module cache..."
	@go clean -modcache
	@echo "Cleaned successfully."

# Lint the code (requires golangci-lint to be installed)
lint:
	@echo "Linting code..."
	@golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@echo "Dependencies installed."

# Update the version in kayveedb.go
update-version-go:
	@sed -i 's/const Version string = "v[0-9]\+\.[0-9]\+\.[0-9]\+"/const Version string = "v$(NEW_VERSION)"/' $(VERSION_GO)
	@echo "Updated version in $(VERSION_GO)"

# Update the version in README.md
update-version-readme:
	@sed -i 's/Current version: \*\*v[0-9]\+\.[0-9]\+\.[0-9]\+\*\*/Current version: \*\*v$(NEW_VERSION)\*\*/' $(README_FILE)
	@echo "Updated version in $(README_FILE)"

# Increment version numbers
increment-patch:
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$(VERSION_MAJOR).$(VERSION_MINOR).$$(( $(VERSION_PATCH) + 1 )) && \
	echo $$NEW_VERSION > $(VERSION_FILE) && \
	$(MAKE) update-version-go NEW_VERSION=$$NEW_VERSION && \
	$(MAKE) update-version-readme NEW_VERSION=$$NEW_VERSION && \
	echo "Version updated to $$NEW_VERSION."

increment-minor:
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$(VERSION_MAJOR).$$(( $(VERSION_MINOR) + 1 )).0 && \
	echo $$NEW_VERSION > $(VERSION_FILE) && \
	$(MAKE) update-version-go NEW_VERSION=$$NEW_VERSION && \
	$(MAKE) update-version-readme NEW_VERSION=$$NEW_VERSION && \
	echo "Version updated to $$NEW_VERSION."

increment-major:
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$$(( $(VERSION_MAJOR) + 1 )).0.0 && \
	echo $$NEW_VERSION > $(VERSION_FILE) && \
	$(MAKE) update-version-go NEW_VERSION=$$NEW_VERSION && \
	$(MAKE) update-version-readme NEW_VERSION=$$NEW_VERSION && \
	echo "Version updated to $$NEW_VERSION."

# Push version to GitHub (for use with your pushversion.sh script)
release: 
	@echo "Pushing new version to GitHub..."
	@./githubBuild/pushversion.sh

# Help message
help:
	@echo "Makefile commands:"
	@echo "  make build            - Prepare the Go package"
	@echo "  make test             - Run tests"
	@echo "  make clean            - Clean Go module cache"
	@echo "  make lint             - Lint the code"
	@echo "  make deps             - Install dependencies"
	@echo "  make increment-patch  - Increment the patch version number"
	@echo "  make increment-minor  - Increment the minor version number"
	@echo "  make increment-major  - Increment the major version number"
	@echo "  make release          - Push new version to GitHub"

.PHONY: all build test clean lint deps update-version-go increment-patch increment-minor increment-major release
