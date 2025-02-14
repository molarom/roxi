run:
	go run main.go

tidy:
	go mod tidy

test:
	CGO_ENABLED=0 go test -v ./...

bench: enwiki-latest-all-titles-in-ns0.gz
	go test -v ./... -bench=. -benchmem

enwiki-latest-all-titles-in-ns0.gz:
	curl https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-all-titles-in-ns0.gz -o enwiki-latest-all-titles-in-ns0.gz
