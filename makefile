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
	mkdir -p build/_output/bin/import-ami
	cp ./images/import-ami/Dockerfile build/_output/bin/import-ami/Dockerfile
	cp ./images/import-ami/entrypoint build/_output/bin/import-ami/entrypoint
	cp ./images/import-ami/user_setup build/_output/bin/import-ami/user_setup
	env GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) go build -i -ldflags="-s -w" -mod=vendor -o build/_output/bin/import-ami/import-ami ./cmd/import-ami

container-build: build
	docker build -t import-ami:latest -f build/_output/bin/import-ami/Dockerfile build/_output/bin/import-ami/

container-push: container-build
	docker image tag import-ami:latest quay.io/dvossel/import-ami:latest
	docker push quay.io/dvossel/import-ami:latest

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
