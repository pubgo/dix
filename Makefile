.PHONY: test, lint
test:
	@go test -v ./dixinternal/... -covermode=count -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run --timeout=10m --verbose

vet:
	go vet ./...
