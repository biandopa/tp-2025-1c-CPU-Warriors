PACKAGES := $(shell go list ./...)

.PHONY: fmt
fmt:
	@echo "==> Formatting code"
	@go fmt $(PACKAGES)
	@echo "==> Done"

.PHONY: imports
imports:
	@echo "==> Fixing imports"
	@goimports -w -l .
	@echo "==> Done"

.PHONY: linter
linter:
	@echo "==> Running linter"
	@golangci-lint run .
	@echo "==> Done"
