.PHONY: test, lint
test:
	@go test -race -v ./... -cover

lint:
	golangci-lint run --timeout=10m --verbose
