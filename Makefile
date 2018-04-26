# golang1.9 or latest
# 1. make help
# 2. make dep
# 3. make build
# ...

SRC := gitlab.33.cn/chain33/chain33/cmd/chain33
SRC_CLI := gitlab.33.cn/chain33/chain33/cmd/cli
SRC_SIGNATORY := gitlab.33.cn/chain33/chain33/cmd/signatory-server
APP := build/chain33
CLI := build/chain33-cli
SIGNATORY := build/signatory-server
LDFLAGS := -ldflags "-w -s"
PKG_LIST := `go list ./... | grep -v "vendor" | grep -v "chain33/test"`
BUILD_FLAGS = -ldflags "-X gitlab.33.cn/chain33/chain33/common/version.GitCommit=`git rev-parse --short=8 HEAD`"
.PHONY: default dep all build release cli linter race test fmt vet bench msan coverage coverhtml docker protobuf clean help

default: build cli

dep: ## Get the dependencies
	@go get -u gopkg.in/alecthomas/gometalinter.v2
	@gometalinter.v2 -i
	@go get -u github.com/mitchellh/gox

all: ## Builds for multiple platforms
	@gox  $(LDFLAGS) $(SRC)
	@cp cmd/chain33/chain33.toml build/
	@mv chain33* build/

build: ## Build the binary file
	@go build $(BUILD_FLAGS) -v -i -o  $(APP) $(SRC)
	@cp cmd/chain33/chain33.toml build/

release: ## Build the binary file
	@go build -v -i -o $(APP) $(LDFLAGS) $(SRC) 
	@cp cmd/chain33/chain33.toml build/

cli: ## Build cli binary
	@go build -v -o $(CLI) $(SRC_CLI)

signatory:
	@go build -v -o $(SIGNATORY) $(SRC_SIGNATORY)
	@cp cmd/signatory-server/signatory.toml build/

build_ci: ## Build the binary file for CI
	@go build -race -v -o $(CLI) $(SRC_CLI)
	@go build  $(BUILD_FLAGS)-race -v -o $(APP) $(SRC)
	@cp cmd/chain33/chain33.toml build/

linter: ## Use gometalinter check code, ignore some unserious warning
	@res=$$(gometalinter.v2 -t --sort=linter --enable-gc --deadline=2m --disable-all \
	--enable=gofmt \
	--enable=gosimple \
	--enable=deadcode \
	--enable=vet \
	--enable=unconvert \
	--enable=interfacer \
	--enable=varcheck \
	--enable=structcheck \
	GS)
#	--enable=staticcheck \
#	--enable=gocyclo \
#	--enable=staticcheck \
#	--enable=golint \
#	--enable=unused \
#	--enable=gotype \
#	--enable=gotypex \
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
	rm -rf build/datadir/addrbook
	rm -rf build/datadir/blockchain.db
	rm -rf build/datadir/mavltree
	rm -rf build/chain33.log

.PHONY: checkgofmt
checkgofmt: ## get all go files and run go fmt on them
	@files=$$(find . -name '*.go' -not -path "./vendor/*" | xargs gofmt -l -s); if [ -n "$$files" ]; then \
		  echo "Error: 'make fmt' needs to be run on:"; \
		  echo "$${files}"; \
		  exit 1; \
		  fi;
