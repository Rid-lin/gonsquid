DOCKER_ACCOUNT = vsvegner

PWD := $(shell pwd)
PROJECTNAME = $(shell basename $(PWD))
PROGRAM_NAME = $(shell basename $(PWD))

VERSION=$(shell git describe --tags)
COMMIT=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
TAG=$(shell git describe --tags |cut -d- -f1)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

PLATFORMS=darwin linux windows
ARCHITECTURES=386 amd64 ppc64

LDFLAGS = -ldflags "-w -s -X=main.Version=${VERSION} -X=main.Build=${COMMIT} -X main.gitTag=${TAG} -X main.gitCommit=${COMMIT} -X main.gitBranch=${BRANCH} -X main.buildTime=${BUILD_TIME}"

# Check for required command tools to build or stop immediately
EXECUTABLES = git go find pwd basename
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH)))

.PHONY: help clean dep build install uninstall

.DEFAULT_GOAL := help

help: ## Display this help screen.
	@echo "Makefile available targets:"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  * \033[36m%-15s\033[0m %s\n", $$1, $$2}'

clean: ## Clean build directory.
	rm -f ./bin/${PROGRAM_NAME}*
#	rmdir ./bin

dep: ## Download the dependencies.
	go mod tidy
	go mod download
	go mod vendor

lint: dep ## Lint the source files
	golangci-lint run --timeout 5m -E golint -e '(struct field|type|method|func) [a-zA-Z`]+ should be [a-zA-Z`]+'
	gosec -quiet ./...

test: dep ## Run tests
	go test -race -p 1 -timeout 300s -coverprofile=.test_coverage.txt ./... && \
    	go tool cover -func=.test_coverage.txt | tail -n1 | awk '{print "Total test coverage: " $$3}'
	@rm .test_coverage.txt

build: dep ## Build program executable for linux platform.
	mkdir -p ./bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${PROGRAM_NAME}_linux_amd64 .

build_ac: dep ## Build program executable for linux platform.
	mkdir -p ./bin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-linkmode external -extldflags '-static' -s -w" -o bin/${PROGRAM_NAME}_$(VERSION)_linux_$(COMMIT)_amd64_cgo .

build_alpine: ## Build program executable for linux platform.
	mkdir -p ./bin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-linkmode external -extldflags '-static' -s -w" -o bin/${PROGRAM_NAME} .

build_a: dep ## Build program executable for linux platform.
	mkdir -p ./bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-linkmode external -extldflags '-static' -s -w" -o bin/${PROGRAM_NAME}_$(VERSION)_linux_$(COMMIT)_amd64 .

build_all: dep ## Build program executable for all platform.
	mkdir -p ./bin
	$(foreach GOOS, $(PLATFORMS),\
	$(foreach GOARCH, $(ARCHITECTURES), $(shell export GOOS=$(GOOS); export GOARCH=$(GOARCH); go build -v $(LDFLAGS) -o ./bin/$(PROJECTNAME)_$(VERSION)_$(GOOS)_$(COMMIT)_$(GOARCH))))

pack: ## Packing all executable files using UPX 
	upx ./bin/*

buildnpack: dep build pack ## Builds the program and packs it with UPX.

buildallnpack: dep build pack ## Builds the program executable for all platform and packs it with UPX.

install: ## Install program executable into /usr/bin directory.
	install -pm 755 bin/${PROGRAM_NAME} /usr/bin/${PROGRAM_NAME}

uninstall: ## Uninstall program executable from /usr/bin directory.
	rm -f /usr/bin/${PROGRAM_NAME}

docker-build: ## Build docker image
	docker build -t ${DOCKER_ACCOUNT}/${PROGRAM_NAME}:${TAG} .
	docker image prune --force --filter label=stage=intermediate

docker-push: ## Push docker image to registry
	docker push ${DOCKER_ACCOUNT}/${PROGRAM_NAME}:${TAG}