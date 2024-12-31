.PHONY: generate test clean build lint format deploy setup

VERSION := $(shell curl -s https://api.github.com/repos/Predixus/DynaRAG/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
ENV_FILE := .env
BINARY_DIR := $(shell pwd)/bin
REQUIRED_BINS := libtokenizers.a onnxruntime.so

setup:
	mkdir -p $(BINARY_DIR)
	for bin in $(REQUIRED_BINS); do \
		curl -L -o $(BINARY_DIR)/$$bin https://github.com/Predixus/DynaRAG/releases/download/$(VERSION)/$$bin; \
	done


generate:
	@# Load .env file and export variables (without modifying .env)
	@if [ -f $(ENV_FILE) ]; then \
		while IFS= read -r line; do \
			[ -z "$$line" ] || [ "$${line#\#}" != "$$line" ] || export "$$line"; \
		done < $(ENV_FILE); \
	fi; \
	go generate ./...


test:
	@# Load .env file and export variables for tests
	@if [ -f $(ENV_FILE) ]; then \
		while IFS= read -r line; do \
			[ -z "$$line" ] || [ "$${line#\#}" != "$$line" ] || export "$$line"; \
		done < $(ENV_FILE); \
	fi; \
	go test ./...


clean:
	@echo "Cleaning build artifacts..."
	@rm -rf ./bin ./build ./dist ./tmp


build:
	@# Load .env file and export variables for build
	@if [ -f $(ENV_FILE) ]; then \
		while IFS= read -r line; do \
			[ -z "$$line" ] || [ "$${line#\#}" != "$$line" ] || export "$$line"; \
		done < $(ENV_FILE); \
	fi; \
	go build -o ./bin/myapp ./...

format:
	@echo "Formatting code..."
	go fmt ./...

