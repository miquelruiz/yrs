.PHONY: all fmt test gen build clean

all: fmt test

fmt:
	go fmt ./...

test: gen
	go test --tags fts5 -v ./...

gen:
	go generate -v ./...

build: gen
	go build --tags fts5 -v ./...

clean:
	rm -rf pkg/yrs/db

update list-channels:
	go run --tags fts5 cmd/yrs/main.go --config config.yml $@
