.PHONY: build build-darwin build-linux build-darwin-arm64

build: build-darwin build-linux build-darwin-arm64

build-darwin:
	env GOOS=darwin GOARCH=amd64 go build -o securityhub-exporter-darwin main.go

build-linux:
	env GOOS=linux GOARCH=amd64 go build -o securityhub-exporter-linux main.go

build-darwin-arm64:
	env GOOS=darwin GOARCH=arm64 go build -o securityhub-exporter-darwin-arm64 main.go

test:
	go test ./...

clean:
	rm -f securityhub-exporter
