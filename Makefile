.PHONY: build build-server build-client clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

build: build-server build-client

build-server:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/usbip-server-darwin-arm64 ./cmd/usbip-server
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/usbip-server-darwin-amd64 ./cmd/usbip-server

build-client:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/usbip-client-linux-amd64 ./cmd/usbip-client
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/usbip-client-linux-arm64 ./cmd/usbip-client

clean:
	rm -rf bin/

test:
	go test -v ./...

fmt:
	go fmt ./...
