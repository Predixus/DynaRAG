.PHONY: generate test test-all clean format 

generate:
	go generate ./...

test:
	go test -v ./... --short

test-all:
	go test -v ./...

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf ./bin ./build ./dist ./tmp
	go clean -testcache

format:
	@echo "Formatting code..."
	go fmt ./...

