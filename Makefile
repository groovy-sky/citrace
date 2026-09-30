BINARY := citrace-shell
VERSION ?= dev

.PHONY: all build test race vet clean release

all: test build

build:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux go build -trimpath \
		-ldflags="-s -w -X main.buildVersion=$(VERSION)" \
		-o dist/$(BINARY) ./cmd/citrace-shell

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

release:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.buildVersion=$(VERSION)" -o dist/$(BINARY)-linux-amd64 ./cmd/citrace-shell
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w -X main.buildVersion=$(VERSION)" -o dist/$(BINARY)-linux-arm64 ./cmd/citrace-shell
	cd dist && sha256sum $(BINARY)-linux-amd64 $(BINARY)-linux-arm64 > SHA256SUMS

clean:
	rm -rf dist