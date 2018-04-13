# golang1.9 or latest
# 1. make help
# 2. make dep
# 3. make build
# ...

APP := build/chain33
CLI := build/chain33-cli
LDFLAGS := -ldflags "-w -s"
PKG_LIST := `go list ./... | grep -v "vendor" | grep -v "chain33/test"`

.PHONY: default dep all build release cli linter race test fmt vet bench msan coverage coverhtml docker protobuf clean help

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

linter: ## Use gometalinter check code, ignore some unserious warning
	@res=$$(gometalinter.v2 --disable-all --enable=errcheck --enable=vet --enable=vetshadow --enable=gofmt --enable=gosimple \
		--enable=deadcode --enable=staticcheck --enable=unused --enable=varcheck --enable=golint --vendor ./... | \
		grep -v "error return value not checked" | \
		grep -v "composite literal uses unkeyed fields" | \
		grep -v "declaration of \"err\" shadows declaration at" | \
		grep -v "should have comment or be unexported" | \
		grep -v "should be of the form" | \
		grep -v "if block ends with a return statement" | \
		grep -v "Id should be ID"); \
	if [ -n "$$res" ]; then \
		echo "$${res}"; \
		exit 1; \
		fi;

race: dep ## Run data race detector
	@go test -race -short ./...

test: ## Run unittests
	@go test -parallel 1 -race $(PKG_LIST)

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

.PHONY: checkgofmt
checkgofmt: ## get all go files and run go fmt on them
	@files=$$(find . -name '*.go' -not -path "./vendor/*" | xargs gofmt -l -s); if [ -n "$$files" ]; then \
		  echo "Error: 'make fmt' needs to be run on:"; \
		  echo "$${files}"; \
		  exit 1; \
		  fi;
