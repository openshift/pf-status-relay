OCI_BIN ?= docker

IMAGE_REGISTRY ?= localhost:5000
IMAGE_NAME ?= pf-status-relay
IMAGE_TAG ?= latest

clean:
	rm -rf bin
	go clean -modcache -testcache

build:
	hack/build.sh

image-build:
	$(OCI_BIN) build -t ${IMAGE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG} -f Dockerfile .

test-build:
	go test -v ./... -count=1

go-lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

go-lint: go-lint-install
	go mod tidy
	go fmt ./...
	golangci-lint run --color always -v ./...
