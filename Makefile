.PHONY: run build fmt test test-integration vet clean

run:
	go run ./cmd/tfguard

build:
	go build -ldflags="-X main.Version=0.1.0" -o bin/tfguard ./cmd/tfguard

fmt:
	go fmt ./...

test:
	go test ./... -cover

# Requires terraform on PATH; runs `terraform init/validate` on testdata/fix cases.
test-integration:
	go test ./internal/fix/ -integration -count=1 -v

vet:
	go vet ./...

clean:
	rm -rf bin
