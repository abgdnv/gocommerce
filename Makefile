# modules is a list of directories containing go.mod files
MODULES := $(shell find . -name "go.mod" -not -path "./vendor/*" -exec dirname {} \;)

.PHONY: help
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: gen
gen: proto sqlc ## Generate all code

.PHONY: proto
proto: ## Generate Go code from Protobuf definitions
	@protoc \
		--proto_path=pkg/api/proto \
		--go_out=pkg/api/gen/go --go_opt=paths=source_relative \
		--go-grpc_out=pkg/api/gen/go --go-grpc_opt=paths=source_relative \
		pkg/api/proto/product/v1/product.proto
	@echo "✅ Protobuf code generated"

.PHONY: sqlc
sqlc: ## Generate sqlc code from SQL
	@sqlc generate -f product_service/internal/store/sqlc.yaml
	@echo "✅ sqlc code for product service generated"
	@sqlc generate -f order_service/internal/store/sqlc.yaml
	@echo "✅ sqlc code for order service generated"

.PHONY: lint
lint: ## Run linter in all modules
	@echo "Running golangci-lint in all modules..."
	@for dir in $(MODULES); do \
		echo "==> Linting $$dir"; \
		(cd "$$dir" && golangci-lint run ./...); \
	done

.PHONY: docker-build
docker-build: ## Build docker images
	@docker compose build

.PHONY: docker-up
docker-up: ## docker compose up -d
	@docker compose up -d

.PHONY: docker-down
docker-down: ## docker compose down
	@docker compose down

.PHONY: test
test: ## Run tests in all modules
	@echo "Running tests in all modules..."
	@for dir in $(MODULES); do \
			echo "==> Testing $$dir"; \
			(cd "$$dir" && go test --count=1 ./...); \
	done

.PHONY: testv
testv: ## Run tests in all modules with verbose output
	@echo "Running tests in all modules..."
	@for dir in $(MODULES); do \
		echo "==> Testing $$dir"; \
		(cd "$$dir" && go test -v --count=1 ./...); \
	done

.PHONY: tidy
tidy: ## Run tests in all modules with verbose output
	@echo "Tidying all Go modules..."
	@for dir in $(MODULES); do \
		echo "==> Tidying module in $$dir"; \
		(cd $$dir && go mod tidy); \
	done

	@echo "Done."
.DEFAULT_GOAL := help
