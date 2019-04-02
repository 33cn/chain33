# golang1.9 or latest
# 1. make help
# 2. make dep
# 3. make build
# ...
SRC := github.com/33cn/chain33/cmd/chain33
SRC_CLI := github.com/33cn/chain33/cmd/cli
SRC_SIGNATORY := github.com/33cn/chain33/cmd/signatory-server
SRC_MINER := github.com/33cn/chain33/cmd/miner_accounts
APP := build/chain33
CLI := build/chain33-cli
SIGNATORY := build/signatory-server
MINER := build/miner_accounts
AUTOTEST := build/autotest/autotest
SRC_AUTOTEST := github.com/33cn/chain33/cmd/autotest
LDFLAGS := -ldflags "-w -s"
PKG_LIST := `go list ./... | grep -v "vendor" | grep -v "mocks"`
PKG_LIST_VET := `go list ./... | grep -v "vendor" | grep -v "common/crypto/sha3" | grep -v "common/log/log15"`
PKG_LIST_INEFFASSIGN= `go list -f {{.Dir}} ./... | grep -v "vendor" | grep -v "common/crypto/sha3" | grep -v "common/log/log15" | grep -v "common/ed25519"`
PKG_LIST_Q := `go list ./... | grep -v "vendor" | grep -v "mocks"`
PKG_LIST_GOSEC := `go list -f "${GOPATH}/src/{{.ImportPath}}" ./... | grep -v "vendor" | grep -v "mocks" | grep -v "cmd" | grep -v "types" | grep -v "commands" | grep -v "log15" | grep -v "ed25519" | grep -v "crypto"`
BUILD_FLAGS = -ldflags "-X github.com/33cn/chain33/common/version.GitCommit=`git rev-parse --short=8 HEAD`"
MKPATH=$(abspath $(lastword $(MAKEFILE_LIST)))
MKDIR=$(dir $(MKPATH))
DAPP := ""
PROJ := "build"
.PHONY: default dep all build release cli linter race test fmt vet bench msan coverage coverhtml docker docker-compose protobuf clean help autotest

default: build cli depends

dep: ## Get the dependencies
	@go get -u gopkg.in/alecthomas/gometalinter.v2
	@gometalinter.v2 -i
	@go get -u github.com/mitchellh/gox
	@go get -u github.com/vektra/mockery/.../
	@go get -u mvdan.cc/sh/cmd/shfmt
	@go get -u mvdan.cc/sh/cmd/gosh
	@apt install clang-format
	@apt install shellcheck

all: ## Builds for multiple platforms
	@gox  $(LDFLAGS) $(SRC)
	@cp cmd/chain33/chain33.toml build/
	@cp cmd/chain33/bityuan.toml build/
	@mv chain33* build/

build: ## Build the binary file
	@go build $(BUILD_FLAGS) -v -i -o  $(APP) $(SRC)
	@cp cmd/chain33/chain33.toml build/
	@cp cmd/chain33/bityuan.toml build/

release: ## Build the binary file
	@go build -v -i -o $(APP) $(LDFLAGS) $(SRC) 
	@cp cmd/chain33/chain33.toml build/
	@cp cmd/chain33/bityuan.toml build/
	@cp cmd/chain33/chain33.para.toml build/

cli: ## Build cli binary
	@go build -v -i -o $(CLI) $(SRC_CLI)

execblock: ## Build cli binary
	@go build -v -i -o build/execblock github.com/33cn/chain33/cmd/execblock


para:
	@go build -v -o build/$(NAME) -ldflags "-X $(SRC_CLI)/buildflags.ParaName=user.p.$(NAME). -X $(SRC_CLI)/buildflags.RPCAddr=http://localhost:8901" $(SRC_CLI)


autotest:## build autotest binary
	@go build -v -i -o $(AUTOTEST) $(SRC_AUTOTEST)
	@if [ -n "$(dapp)" ]; then \
		cd build/autotest && bash ./copy-autotest.sh local && cd local && bash ./local-autotest.sh $(dapp) && cd ../../../; \
	fi
autotest_ci: autotest ## autotest jerkins ci
	@cd build/autotest && bash ./copy-autotest.sh jerkinsci/temp$(proj) && cd jerkinsci && bash ./jerkins-ci-autotest.sh $(proj) && cd ../../../

signatory:
	@cd cmd/signatory-server/signatory && bash ./create_protobuf.sh && cd ../.../..
	@go build -v -o $(SIGNATORY) $(SRC_SIGNATORY)
	@cp cmd/signatory-server/signatory.toml build/

miner:
	@cd cmd/miner_accounts/accounts && bash ./create_protobuf.sh && cd ../.../..
	@go build -v -o $(MINER) $(SRC_MINER)
	@cp cmd/miner_accounts/miner_accounts.toml build/

build_ci: depends ## Build the binary file for CI
	@go build -v -i -o $(CLI) $(SRC_CLI)
	@go build  $(BUILD_FLAGS) -v -o $(APP) $(SRC)
	@cp cmd/chain33/chain33.toml build/

linter: vet ineffassign gosec ## Use gometalinter check code, ignore some unserious warning
	@./golinter.sh "filter"
	@find . -name '*.sh' -not -path "./vendor/*" | xargs shellcheck

linter_test: ## Use gometalinter check code, for local test
	@./golinter.sh "test" "${p}"
	@find . -name '*.sh' -not -path "./vendor/*" | xargs shellcheck

gosec:
	@gosec -quiet=true -exclude=G107,G402,G302 ${PKG_LIST_GOSEC}

race: ## Run data race detector
	@go test -race -short $(PKG_LIST)

vet:
	@go vet ${PKG_LIST_VET}

ineffassign:
	@ineffassign -n ${PKG_LIST_INEFFASSIGN}

test: ## Run unittests
	@go test -race $(PKG_LIST)

testq: ## Run unittests
	@go test $(PKG_LIST_Q)

fmt: fmt_proto fmt_shell ## go fmt
	go fmt ./...
	find . -name '*.go' -not -path "./vendor/*" | xargs goimports -l -w

.PHONY: fmt_proto fmt_shell
fmt_proto: ## go fmt protobuf file
	@find . -name '*.proto' -not -path "./vendor/*" | xargs clang-format -i

fmt_shell: ## check shell file
	find . -name '*.sh' -not -path "./vendor/*" | xargs shfmt -w -s -i 4 -ci -bn

bench: ## Run benchmark of all
	@go test ./... -v -bench=.

msan: ## Run memory sanitizer
	@go test -msan -short $(PKG_LIST)

coverage: ## Generate global code coverage report
	@./build/tools/coverage.sh

coverhtml: ## Generate global code coverage report in HTML
	@./build/tools/coverage.sh html

docker: ## build docker image for chain33 run
	@sudo docker build . -f ./build/Dockerfile-run -t chain33:latest

docker-compose: ## build docker-compose for chain33 run
	@cd build && if ! [ -d ci ]; then \
	 make -C ../ ; \
	 fi; \
	 cp chain33* Dockerfile  docker-compose* ci/ && cd ci/ && ./docker-compose-pre.sh run $(PROJ) $(DAPP)  && cd ../..

docker-compose-down: ## build docker-compose for chain33 run
	@cd build && if [ -d ci ]; then \
	 cp chain33* Dockerfile  docker-compose* ci/ && cd ci/ && ./docker-compose-pre.sh down $(PROJ) $(DAPP) && cd .. ; \
	 fi; \
	 cd ..

fork-test: ## build fork-test for chain33 run
	@cd build && cp chain33* Dockerfile system-fork-test.sh docker-compose* ci/ && cd ci/ && ./docker-compose-pre.sh forktest $(PROJ) $(DAPP) && cd ../..


clean: ## Remove previous build
	@rm -rf $(shell find . -name 'datadir' -not -path "./vendor/*")
	@rm -rf build/chain33*
	@rm -rf build/relayd*
	@rm -rf build/*.log
	@rm -rf build/logs
	@rm -rf build/autotest/autotest
	@rm -rf build/ci
	@go clean

proto:protobuf

protobuf: ## Generate protbuf file of types package
	@cd types/proto && ./create_protobuf.sh && cd ../..
	@find ./system/dapp ./system/store/mavl -maxdepth 2 -type d  -name proto -exec make -C {} \;

depends: ## Generate depends file of types package
	@find ./system/dapp -maxdepth 2 -type d  -name cmd -exec make -C {} OUT="$(MKDIR)build/ci" FLAG= \;

help: ## Display this help screen
	@printf "Help doc:\nUsage: make [command]\n"
	@printf "[command]\n"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	
cleandata:
	rm -rf build/datadir/addrbook
	rm -rf build/datadir/blockchain.db
	rm -rf build/datadir/mavltree
	rm -rf build/chain33.log

fmt_go: fmt_shell ## go fmt
	@go fmt ./...
	@find . -name '*.go' -not -path "./vendor/*" | xargs goimports -l -w


.PHONY: checkgofmt
checkgofmt: ## get all go files and run go fmt on them
	@files=$$(find . -name '*.go' -not -path "./vendor/*" | xargs gofmt -l -s); if [ -n "$$files" ]; then \
		  echo "Error: 'make fmt' needs to be run on:"; \
		  find . -name '*.go' -not -path "./vendor/*" | xargs gofmt -l -s ;\
		  exit 1; \
		  fi;
	@files=$$(find . -name '*.go' -not -path "./vendor/*" | xargs goimports -l -w); if [ -n "$$files" ]; then \
		  echo "Error: 'make fmt' needs to be run on:"; \
		  find . -name '*.go' -not -path "./vendor/*" | xargs goimports -l -w ;\
		  exit 1; \
		  fi;

.PHONY: mock
mock:
	@cd client && mockery -name=QueueProtocolAPI && mv mocks/QueueProtocolAPI.go mocks/api.go && cd -
	@cd queue && mockery -name=Client && mv mocks/Client.go mocks/client.go && cd -
	@cd common/db && mockery -name=KV && mv mocks/KV.go mocks/kv.go && cd -
	@cd common/db && mockery -name=KVDB && mv mocks/KVDB.go mocks/kvdb.go && cd -
	@cd types/ && mockery -name=Chain33Client && mv mocks/Chain33Client.go mocks/chain33client.go && cd -


.PHONY: auto_ci_before auto_ci_after auto_ci
auto_ci_before: clean fmt protobuf mock
	@echo "auto_ci"
	@go version
	@protoc --version
	@mockery -version
	@docker version
	@docker-compose version
	@git version
	@git status

.PHONY: auto_ci_after
auto_ci_after: clean fmt protobuf mock
	@git add *.go *.sh *.proto
	@git status
	@files=$$(git status -suno);if [ -n "$$files" ]; then \
		  git add *.go *.sh *.proto; \
		  git status; \
		  git commit -m "auto ci [ci-skip]"; \
		  git push origin HEAD:$(branch); \
		  fi;

.PHONY: auto_ci
auto_fmt := find . -name '*.go' -not -path './vendor/*' | xargs goimports -l -w
auto_ci: clean fmt_proto fmt_shell protobuf mock
	@-find . -name '*.go' -not -path './vendor/*' | xargs gofmt -l -w -s
	@-${auto_fmt}
	@-find . -name '*.go' -not -path './vendor/*' | xargs gofmt -l -w -s
	@${auto_fmt}
	@git status
	@files=$$(git status -suno);if [ -n "$$files" ]; then \
		  git add *.go *.sh *.proto; \
		  git status; \
		  git commit -a -m "auto ci"; \
		  git push origin HEAD:$(branch); \
		  exit 1; \
		  fi;

webhook_auto_ci: clean fmt_proto fmt_shell protobuf mock
	@-find . -name '*.go' -not -path './vendor/*' | xargs gofmt -l -w -s
	@-${auto_fmt}
	@-find . -name '*.go' -not -path './vendor/*' | xargs gofmt -l -w -s
	@${auto_fmt}
	@git status
	@files=$$(git status -suno);if [ -n "$$files" ]; then \
		  git status; \
		  git commit -a -m "auto ci"; \
		  git push origin ${b}; \
		  exit 0; \
		  fi;

addupstream:
	git remote add upstream https://github.com/33cn/chain33.git
	git remote -v

sync:
	git fetch upstream
	git checkout master
	git merge upstream/master
	git push origin master

branch:
	make sync
	git checkout -b ${b}

push:
	@if [ -n "$$m" ]; then \
	git commit -a -m "${m}" ; \
	fi;
	make sync
	git checkout ${b}
	git merge master
	git push origin ${b}

pull:
	@remotelist=$$(git remote | grep ${name});if [ -z $$remotelist ]; then \
		echo ${remotelist}; \
		git remote add ${name} https://github.com/${name}/chain33.git ; \
	fi;
	git fetch ${name}
	git checkout ${name}/${b}
	git checkout -b ${name}-${b}
pullsync:
	git fetch ${name}
	git checkout ${name}-${b}
	git merge ${name}/${b}
pullpush:
	@if [ -n "$$m" ]; then \
	git commit -a -m "${m}" ; \
	fi;
	make pullsync
	git push ${name} ${name}-${b}:${b}

webhook:
	git checkout ${b}
	make webhook_auto_ci name=${name} b=${b}
