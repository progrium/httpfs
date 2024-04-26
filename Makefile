.PHONY: httpfs release

VERSION=0.2dev

httpfs:
	go build -ldflags="-X 'main.Version=${VERSION}'" .

release:
	VERSION=$(VERSION) goreleaser release --snapshot --clean
	./local/macsign/notarize