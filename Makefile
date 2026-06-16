.PHONY: build test docker-build docker-push install clean

# Version
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Docker
DOCKER_REPO ?= aivar/safeupgrade
DOCKER_TAG ?= latest

# Build info
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.BuildTime=$(BUILD_TIME)

build:
	@echo "Building SafeUpgrade $(VERSION)..."
	go build -ldflags "$(LDFLAGS)" -o safeupgrade .

build-all:
	@echo "Building for all platforms..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/safeupgrade-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/safeupgrade-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/safeupgrade-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/safeupgrade-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/safeupgrade-windows-amd64.exe .

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

coverage:
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html

lint:
	@echo "Running linters..."
	golangci-lint run

docker-build:
	@echo "Building Docker image $(DOCKER_REPO):$(DOCKER_TAG)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_REPO):$(DOCKER_TAG) \
		-t $(DOCKER_REPO):$(VERSION) \
		.

docker-build-multiarch:
	@echo "Building multi-architecture Docker images..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_REPO):$(DOCKER_TAG) \
		-t $(DOCKER_REPO):$(VERSION) \
		--push \
		.

docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_REPO):$(DOCKER_TAG)
	docker push $(DOCKER_REPO):$(VERSION)

install:
	@echo "Installing SafeUpgrade..."
	go install -ldflags "$(LDFLAGS)" .

clean:
	@echo "Cleaning..."
	rm -rf safeupgrade dist/ coverage.out coverage.html
	rm -rf scan_report.json upgrade_report.json

run-sample:
	@echo "Running on sample project..."
	./safeupgrade scan --repo testdata/sample-project

run-upgrade-sample:
	@echo "Running upgrade on sample project..."
	./safeupgrade upgrade --repo testdata/sample-project --policy configs/policy.yaml

dev:
	@echo "Starting development environment..."
	docker-compose up -d postgres redis
	@echo "Run 'make watch' in another terminal for live reload"

watch:
	@echo "Starting live reload..."
	air

stop-dev:
	@echo "Stopping development environment..."
	docker-compose down

# Release workflow
release: test lint build-all
	@echo "Creating release $(VERSION)..."
	gh release create $(VERSION) dist/* \
		--title "Release $(VERSION)" \
		--notes "See CHANGELOG.md for details"

# Helm chart
helm-package:
	@echo "Packaging Helm chart..."
	helm package helm/safeupgrade -d dist/

helm-install:
	@echo "Installing Helm chart..."
	helm install safeupgrade helm/safeupgrade \
		--namespace safeupgrade \
		--create-namespace

# Documentation
docs-serve:
	@echo "Serving documentation..."
	cd docs && mkdocs serve

docs-build:
	@echo "Building documentation..."
	cd docs && mkdocs build

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@echo "Reverting database migrations..."
	migrate -path migrations -database "$(DATABASE_URL)" down 1

# Benchmark
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Security scan
security-scan:
	@echo "Running security scan..."
	gosec ./...
	trivy fs .

help:
	@echo "SafeUpgrade Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build              Build binary"
	@echo "  make test               Run tests"
	@echo "  make docker-build       Build Docker image"
	@echo "  make docker-push        Push Docker image"
	@echo "  make install            Install binary"
	@echo "  make release            Create a release"
	@echo "  make dev                Start dev environment"
	@echo "  make help               Show this help"
