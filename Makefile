.PHONY: httpfs release

VERSION=0.1dev

httpfs:
	go build -ldflags="-X 'main.Version=${VERSION}'" .

release:
	VERSION=$(VERSION) goreleaser release --snapshot --clean
	./local/macsign/notarize