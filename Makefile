SHELL := /bin/bash

.PHONY: build clean validate watch

build:
	@bash build.sh

clean:
	@bash clean.sh

validate:
	@bash validate.sh

watch:
	@echo "Watching src/ for changes..."
	@fswatch -o src/ | xargs -n1 -I{} make build
