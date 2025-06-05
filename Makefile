.PHONY: test
test:
	@go test -race -v ./... -cover

lint:
	golangci-lint run --out-format=colored-line-number --timeout=10m --verbose
