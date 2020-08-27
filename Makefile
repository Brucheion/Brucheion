GO=go
NPM=cd ui && npm
BIN=Brucheion
VERSION=$$(git describe --abbrev=0 --tags)
NODE_MODULES=ui/node_modules

.PHONY: all dev-ui build build-release build-ui test clean deps pkged.go

all: deps test build

build: build-ui brucheion

pkged.go:
	pkger -exclude image_archive

brucheion: pkged.go
	$(GO) build -o $(BIN) -v

build-release: deps test build-ui pkged.go
	env GOOS=darwin  GOARCH=amd64 $(GO) build -o "release/${BIN}-${VERSION}-macos-x86_64"
	env GOOS=windows GOARCH=386   $(GO) build -o "release/${BIN}-${VERSION}-i386.exe"
	env GOOS=windows GOARCH=amd64 $(GO) build -o "release/${BIN}-${VERSION}-x86_64.exe"

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
