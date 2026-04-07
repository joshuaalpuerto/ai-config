SHELL := /bin/bash

.PHONY: build clean validate watch install

install:
	@go install ./cmd/aicfg

build:
	@go run ./cmd/aicfg build

clean:
	@go run ./cmd/aicfg clean

validate:
	@go run ./cmd/aicfg validate

watch:
	@echo "Watching src/ for changes..."
	@fswatch -o src/ | xargs -n1 -I{} make build
