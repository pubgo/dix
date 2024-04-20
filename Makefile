.PHONY: test
test:
	@go test -race -v ./... -cover

.PHONY: vet
vet:
	@go vet ./...
