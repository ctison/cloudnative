.PHONY: lint test build release clean

SHELL := bash --noprofile --norc -O nullglob -euo pipefail

lint:
	golangci-lint run

test:
	go test -race

build:
	goreleaser build --rm-dist --snapshot

release:
	goreleaser release --rm-dist --skip-publish --snapshot

clean:
	rm -rf -- dist/