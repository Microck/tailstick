GO ?= go

.PHONY: fmt test vet build build-all sandbox-linux icons

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

build:
	$(GO) build ./cmd/tailstick-linux-cli
	$(GO) build ./cmd/tailstick-linux-gui
	$(GO) build ./cmd/tailstick-windows-cli
	$(GO) build ./cmd/tailstick-windows-gui

build-all:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/tailstick-linux-cli ./cmd/tailstick-linux-cli
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/tailstick-linux-gui ./cmd/tailstick-linux-gui
	GOOS=windows GOARCH=amd64 $(GO) build -o dist/tailstick-windows-cli.exe ./cmd/tailstick-windows-cli
	GOOS=windows GOARCH=amd64 $(GO) build -o dist/tailstick-windows-gui.exe ./cmd/tailstick-windows-gui

icons:
	./scripts/generate-windows-icon.sh

sandbox-linux:
	mkdir -p dist
	GOOS=linux $(GO) build -o dist/tailstick-linux-cli ./cmd/tailstick-linux-cli
	chmod +x tests/sandbox/linux-sandbox-e2e.sh
	docker run --rm -v "$(CURDIR):/src" -w /src ubuntu:24.04 /src/tests/sandbox/linux-sandbox-e2e.sh
