# CheckLogs Go SDK Makefile

.PHONY: help build test lint clean run-example install-deps update-deps benchmark

# Default target
help:
	@echo "CheckLogs Go SDK - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make install-deps    Install dependencies"
	@echo "  make update-deps     Update dependencies"
	@echo "  make build          Build the project"
	@echo "  make test           Run tests"
	@echo "  make test-verbose   Run tests with verbose output"
	@echo "  make test-coverage  Run tests with coverage"
	@echo "  make lint           Run linting"
	@echo "  make fmt            Format code"
	@echo ""
	@echo "Examples:"
	@echo "  make run-example    Run the basic example"
	@echo "  make run-benchmark  Run performance benchmark"
	@echo "  make run-stress     Run stress test"
	@echo ""
	@echo "Release:"
	@echo "  make clean          Clean build artifacts"
	@echo "  make release        Prepare for release"
	@echo ""
	@echo "Environment Variables:"
	@echo "  CHECKLOGS_API_KEY   Your CheckLogs API key (required for examples)"

# Install dependencies
install-deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy

# Update dependencies
update-deps:
	@echo "🔄 Updating dependencies..."
	go get -u ./...
	go mod tidy

# Build the project
build:
	@echo "🔨 Building project..."
	go build ./...

# Run tests
test:
	@echo "🧪 Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linting
lint:
	@echo "🔍 Running linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# Format code
fmt:
	@echo "📝 Formatting code..."
	go fmt ./...
	goimports -w .

# Run the basic example
run-example:
	@echo "🚀 Running basic example..."
	@if [ -z "$(CHECKLOGS_API_KEY)" ]; then \
		echo "❌ CHECKLOGS_API_KEY environment variable is required"; \
		echo "   Set it with: export CHECKLOGS_API_KEY=your-api-key"; \
		exit 1; \
	fi
	cd examples && go run basic.go

# Run performance benchmark
run-benchmark:
	@echo "⚡ Running benchmark..."
	@if [ -z "$(CHECKLOGS_API_KEY)" ]; then \
		echo "❌ CHECKLOGS_API_KEY environment variable is required"; \
		exit 1; \
	fi
	cd examples && go run basic.go benchmark 1000

# Run stress test
run-stress:
	@echo "💪 Running stress test..."
	@if [ -z "$(CHECKLOGS_API_KEY)" ]; then \
		echo "❌ CHECKLOGS_API_KEY environment variable is required"; \
		exit 1; \
	fi
	cd examples && go run basic.go stress

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	go clean
	rm -f coverage.out coverage.html
	rm -rf dist/

# Prepare for release
release: clean lint test
	@echo "🚀 Preparing for release..."
	@echo "✅ All checks passed!"

# Development setup
setup: install-deps
	@echo "🛠️  Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@echo "✅ Development environment ready!"

# Quick development cycle
dev: fmt lint test
	@echo "✅ Quick development cycle completed!"

# Benchmark tests
benchmark:
	@echo "📊 Running Go benchmarks..."
	go test -bench=. -benchmem ./...

# Security check
security:
	@echo "🔒 Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Dependency vulnerability check
vuln-check:
	@echo "🔍 Checking for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi

# Complete CI pipeline
ci: setup lint security vuln-check test-coverage
	@echo "🎉 CI pipeline completed successfully!"

# Local development with file watching (requires fswatch or inotify-tools)
watch:
	@echo "👀 Watching for file changes..."
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -o . | while read f; do make dev; done; \
	elif command -v inotifywait >/dev/null 2>&1; then \
		while inotifywait -r -e modify,create,delete .; do make dev; done; \
	else \
		echo "❌ Neither fswatch nor inotifywait found. Please install one for file watching."; \
		exit 1; \
	fi

# Generate documentation
docs:
	@echo "📚 Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed. Installing..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	fi

# Version information
version:
	@echo "Go version: $$(go version)"
	@echo "Module: $$(go list -m)"
	@echo "Dependencies:"
	@go list -m all

# Example with specific API key (for testing)
example-with-key:
	@read -p "Enter your CheckLogs API key: " api_key; \
	CHECKLOGS_API_KEY=$$api_key make run-example