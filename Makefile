VERSION=$(shell cat VERSION | tr -d '\n')
SHORTSHA=$(shell git rev-parse --short HEAD)
LDFLAGS=-X main.appVersion=$(VERSION) -X main.shortSha=$(SHORTSHA)

.PHONY: build run test

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o mikrotik-exporter .

run: build
	./mikrotik-exporter $(ARGS)

test:
	go test ./...
