.PHONY: all fmt test build install

all: fmt test

fmt:
	go fmt ./...

test:
	go test --tags fts5 -v ./...

build:
	go build --tags fts5 -v ./...

update list-channels:
	go run --tags fts5 ./cmd/yrs --config config.yml $@

install:
	go install --tags fts5 -ldflags="-X 'github.com/miquelruiz/yrs/internal/vcs.Version=$(shell git describe)'" ./cmd/yrs
