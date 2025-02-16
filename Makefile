deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
	go install mvdan.cc/gofumpt@latest

fmt:
	gofumpt -l -w

ci-lint:
	golangci-lint run

lint: fmt ci-lint

test:
	CGO_ENABLED=0 go test .

test-race:
	CGO_ENABLED=1 go test -race .

bench:
	CGO_ENABLED=0 go test -bench=. -benchmem

cover:
	CGO_ENABLED=0 go test -cover
