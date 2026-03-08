SHELL := /bin/bash
SCRIPTS_DIR := scripts

.PHONY: build clean validate watch

build:
	@bash $(SCRIPTS_DIR)/build.sh

clean:
	@bash $(SCRIPTS_DIR)/clean.sh

validate:
	@bash $(SCRIPTS_DIR)/validate.sh

watch:
	@echo "Watching src/ for changes..."
	@fswatch -o src/ | xargs -n1 -I{} make build
