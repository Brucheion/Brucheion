NODE_MODULES=ui/node_modules

GO=go
NPM=cd ui && npm
BIN=Brucheion
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: all
all: deps test build

.PHONY: build
build: ui brucheion

.PHONY: brucheion
brucheion:
	$(GO) build -o $(BIN) -v

.PHONY: ui
ui:
	$(NPM) run build

.PHONY: clean
test:
	$(GO) test -v ./...
	cd ui && npm test

.PHONY: clean
clean:
	$(GO) clean
	rm -f $(BIN)
	rm -r $(NODE_MODULES)

.PHONY: dev
dev:
	$(NPM) run dev

.PHONY: deps
deps: $(NODE_MODULES)

$(NODE_MODULES): ui/package.json ui/package-lock.json
	$(NPM) install
