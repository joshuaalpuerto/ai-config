SHELL := /bin/bash

.PHONY: build clean validate watch install

install:
	@go install ./cmd/...

build:
	@go run ./cmd build

clean:
	@go run ./cmd clean

validate:
	@go run ./cmd validate

watch:
	@echo "Watching src/ for changes..."
	@fswatch -o src/ | xargs -n1 -I{} make build
