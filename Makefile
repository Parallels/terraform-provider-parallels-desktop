
GOX = gox
GOLANGCI_LINT = GOLANGCI_LINT
GOLANGCI_LINT_VERSION ?= v2.14.0
GOSEC = gosec
GOSEC_VERSION ?= v2.29.1-0.20261012130337-71577268350e
TFPROVIDERLINT = tfproviderlint
DEPS_TOOLS = deps_tools
GOFUMPT = gofumpt
TFDOCS-PLUGIN = tfdocs-plugin
MISSPELL = misspell

DEVELOPMENT_TOOLS = $(GOX) $(GOLANGCI_LINT) $(TFPROVIDERLINT) $(GOFUMPT) $(TFDOCS-PLUGIN) $(MISSPELL)
SECURITY_TOOLS = $(GOSEC)

MARKDOWNLINT_IMG := 06kellyjac/markdownlint-cli
MARKDOWNLINT_TAG := 0.28.1

.PHONY: default
default: build

.PHONY: build
build: format docs-check
	# trying to copy .goreleaser.yaml
	go build -a -ldflags '-s -extldflags "-static"'


.PHONY: clean
clean:
	rm -f terraform-provider-parallels-desktop

.PHONY: check-release
check-release: lint format-check build unit-test clean

.PHONY: lint
lint:
	golangci-lint run \
		--timeout 15m0s

.PHONY: format
format:
	gofmt -l -w .

.PHONY: format-check
format-check:
	@echo "Checking formatting..."
	@! gofmt -l . | read

.PHONY: deps
deps: $(DEVELOPMENT_TOOLS) $(SECURITY_TOOLS)

.PHONY: test
test: lint unit-test

.PHONY: unit-test
unit-test:
	go test -v --tags=unit ./$(PKG_NAME)

.PHONY: docs
docs:
	# generate the documents
	cd tools; go generate ./...

.PHONY: docs-lint
docs-lint:
	@echo "Running markdown linter for the documents"
	docker run --rm \
		-v $$(pwd):/markdown:ro \
		$(MARKDOWNLINT_IMG):$(MARKDOWNLINT_TAG) \
		--config .markdownlint.yml \
		docs

.PHONY: docs-misspell
docs-misspell:
	@echo "Running misspell for the documents"
	@misspell -error -source text docs/

.PHONY: docs-check
docs-check: docs-lint docs-misspell

.PHONY: sweep
sweep:
	# sweep cleans the test infra from your account
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test -v ./$(PKG_NAME) -sweep=$(SWEEP) $(ARGS)

.PHONY: security-check
security-check: $(SECURITY_TOOLS)
	# run security checks
	GO111MODULE="on" && gosec -exclude=G104,G106,G115,G402 ./...

$(GOX):
	@echo "Installing gox..."
	@go install github.com/mitchellh/gox@latest

$(GOLANGCI_LINT):
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOSEC):
	@echo "Installing gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

$(GOFUMPT):
	@echo "Installing gofumpt..."
	@go install mvdan.cc/gofumpt@latest

$(TFPROVIDERLINT):
	@echo "Installing tfproviderlint..."
	@go install github.com/bflad/tfproviderlint/cmd/tfproviderlintx@latest

$(TFDOCS-PLUGIN):
	@echo "Installing tfdocs-plugin..."
	@go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

$(MISSPELL):
	@echo "Installing misspell..."
	@go install github.com/client9/misspell/cmd/misspell@latest

$(DEPS_TOOLS):
	@echo "Installing Terraform Provider tools..."
	@cd tools && go get github.com/bflad/tfproviderlint/cmd/tfproviderlint
	@cd tools && go get github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@cd tools && go mod tidy
	@go generate -tags tools tools/tools.go
