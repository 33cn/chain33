# golang1.9 or latest
# 1. make help
# 2. make dep
# 3. make build
# ...

APP := build/chain33
CLI := build/chain33-cli
LDFLAGS := -ldflags "-w -s"
PKG_LIST := $(shell go list ./... | grep -v /vendor/)

.PHONY: default dep all build release cli linter lint race test fmt vet bench msan coverage coverhtml docker protobuf clean help

default: build

dep: ## Get the dependencies
	@go get -u gopkg.in/alecthomas/gometalinter.v2
	@gometalinter.v2 -i
	@go get -u github.com/mitchellh/gox

all: ## Builds for multiple platforms
	@gox $(LDFLAGS)
	@mv chain33* build/

ticket:
	go build -i -v -o chain33
	./chain33 -f chain33.test.toml

build: ## Build the binary file
	@go build -race -v -o $(APP)
	@cp chain33.toml build/

release: ## Build the binary file
	@go build -race -v -o $(APP) $(LDFLAGS)
	@cp chain33.toml build/

cli: ## Build cli binary
	@go build -race -v -o $(CLI) cli/cli.go

linter: ## Use gometalinter check code
	@gometalinter.v2 --disable-all --enable=errcheck --enable=vet --enable=vetshadow --enable=gofmt --enable=gosimple \
	--enable=deadcode --enable=staticcheck --enable=unused --enable=varcheck --vendor ./...

lint: ## Lint the files
	@res=$$(golint -set_exit_status ${PKG_LIST} | grep -v "should have comment or be unexported" | \
		grep -v "should be of the form" | grep -v "if block ends with a return statement"); \
	if [ -n "$$res" ]; then \
		echo "$${res}"; \
		exit 1; \
		fi;

race: dep ## Run data race detector
	@go test -race -short ./...

test: ## Run unittests
	@go test -parallel 1 -race $(shell go list ./... | grep -v "vendor" | grep -v "chain33/test")

fmt: ## go fmt
	@go fmt ./...

vet: ## go vet
	@go vet ./...

bench: ## Run benchmark of all
	@go test ./... -v -bench=.

msan: dep ## Run memory sanitizer
	@go test -msan -short ./...

coverage: ## Generate global code coverage report
	@./build/tools/coverage.sh;

coverhtml: ## Generate global code coverage report in HTML
	@./build/tools/coverage.sh html;

docker: ## build docker image for chain33 run
	@sudo docker build . -f ./build/Dockerfile-run -t chain33:latest

DATADIR:=find . -name 'datadir' -not -path "./vendor/*"
clean: ## Remove previous build
	@rm -rf $(shell find . -name 'datadir' -not -path "./vendor/*")
	@rm -rf build/chain33*
	@rm -rf build/*.log
	@rm -rf build/logs
	@go clean

protobuf: ## Generate protbuf file of types package
	@cd types && ./create_protobuf.sh && cd ..

help: ## Display this help screen
	@printf "Help doc:\nUsage: make [command]\n"
	@printf "[command]\n"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	
cleandata:
	rm -rf datadir/addrbook
	rm -rf datadir/blockchain.db
	rm -rf datadir/mavltree
	rm -rf chain33.log

GOFILES := find . -name '*.go' -not -path "./vendor/*"

.PHONY: checkgofmt
checkgofmt: ## get all go files and run go fmt on them
	@files=$$($(GOFILES) | xargs gofmt -l); if [ -n "$$files" ]; then \
		  echo "Error: 'make fmt' needs to be run on:"; \
		  echo "$${files}"; \
		  exit 1; \
		  fi;
