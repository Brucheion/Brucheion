GO=go
NPM=cd ui && npm
BIN=Brucheion

NODE_MODULES=ui/node_modules

.PHONY: all dev-ui build build-ui test clean deps

all: deps test build

build: build-ui brucheion

brucheion:
	pkger -exclude image_archive
	$(GO) build -o $(BIN) -v

build-ui:
	$(NPM) run build

test:
	$(GO) test -v ./...
	cd ui && npm test

clean:
	$(GO) clean
	rm -f $(BIN)
	rm -r $(NODE_MODULES)

dev-ui:
	$(NPM) run dev

deps: $(NODE_MODULES)

$(NODE_MODULES): ui/package.json ui/package-lock.json
	$(NPM) install
