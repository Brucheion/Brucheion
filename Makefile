GO=go
NPM=cd ui && npm
BIN=Brucheion
NODE_MODULES=ui/node_modules

.PHONY: all app-dev build release app test clean deps

all: deps test build

build: app brucheion

brucheion:
	$(GO) build -o $(BIN) -v

release: deps test app
	./scripts/release.sh

app:
	$(NPM) run build

test:
	$(GO) test -v ./...
	cd ui && npm test

clean:
	$(GO) clean
	rm -f $(BIN)
	rm -r $(NODE_MODULES)

app-dev:
	$(NPM) run dev

deps: $(NODE_MODULES)

$(NODE_MODULES): ui/package.json ui/package-lock.json
	$(NPM) install
