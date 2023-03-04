.PHONY: all fmt test gen build clean

all: fmt test

fmt:
	go fmt ./...

test: gen
	go test -v ./...

gen:
	go generate -v ./...

build: gen
	go build -v ./...

clean:
	rm -rf pkg/yrs/db
