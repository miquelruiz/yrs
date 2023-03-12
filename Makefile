.PHONY: all fmt test gen build clean

all: fmt test

fmt:
	go fmt ./...

test:
	go test --tags fts5 -v ./...

build:
	go build --tags fts5 -v ./...

clean:
	rm -rf pkg/yrs/db

update list-channels:
	go run --tags fts5 cmd/yrs/main.go --config config.yml $@
