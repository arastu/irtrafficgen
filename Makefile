.PHONY: build test vet check-assets
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -s -w -X github.com/arastu/irtrafficgen/internal/version.Version=$(VERSION) -X github.com/arastu/irtrafficgen/internal/version.Commit=$(COMMIT)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o irtrafficgen .

test:
	go test ./...

vet:
	go vet ./...

check-assets:
	test -s internal/assets/geosite.dat
	test -s internal/assets/geoip.dat

all: check-assets vet test build
