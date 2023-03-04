.PHONY: all fmt test

all: fmt test

fmt:
	go fmt ./...

test:
	go test -v ./...
