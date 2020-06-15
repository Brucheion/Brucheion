GO=go
NPM=cd ui && npm
BIN=Brucheion

NODE_MODULES=ui/node_modules

all: deps test build

build:
	$(NPM) run build
	$(GO) build -o $(BIN) -v

test:
	$(GO) test -v ./...
	cd ui && npm test

clean:
	$(GO) clean
	rm -f $(BIN)
	rm -r $(NODE_MODULES)

deps: ui/package.json ui/package-lock.json
ui/package.json:
	$(NPM) install
ui/package-lock.json:
	$(NPM) install
