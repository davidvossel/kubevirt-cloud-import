.PHONY: build \
	unit-test \
	gofmt \
	golint \
	govet \
	deps-update

TARGET_GOOS ?= "linux"
TARGET_GOARCH ?= "amd64"

# Export GO111MODULE=on to enable project to be built from within GOPATH/src
export GO111MODULE=on

build:
	@echo "Building binary"
	mkdir -p build/_output/bin
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/import-ami ./cmd/import-ami

deps-update:
	go mod tidy && \
	go mod vendor

gofmt:
	@echo "Running gofmt"
	gofmt -s -l `find . -path ./vendor -prune -o -type f -name '*.go' -print`

golint:
	@echo "Running go lint"
	hack/lint.sh

govet:
	@echo "Running go vet"
	go vet ./...


unit-test: gofmt govet golint
	go test -v ./...
