PKG_LIST := $(shell go list ./... | grep -v /vendor/)

.PHONY: build test coverage

build:
	@CGO_ENABLED=1 go build -o ./dist/app ./cmd

test:
	@go test -count=1 -v ${PKG_LIST}

coverage:
	@./coverage.sh

dependencies:
	@go get -v ./...