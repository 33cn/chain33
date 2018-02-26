LDFLAGS = -ldflags "-w -s"
APP = chain33

.PHONY: build

default: build

setup:
	@go get -u gopkg.in/alecthomas/gometalinter.v2
	@gometalinter.v2 -i

build:
	@go build -o $(APP)

release:
	@go build -o $(APP) $(LDFLAGS)

linter:
	@gometalinter.v2 --disable-all --enable=errcheck --enable=vet --enable=vetshadow --enable=gofmt --enable=gosimple \
	--enable=deadcode --enable=staticcheck --enable=unused --enable=varcheck  $(go list ./... | grep -v /vendor/)

race:
	@go test  -v -race ./...

test:
	@go test -v ./...

cover:
	@go test -race -v -coverprofile=coverage.out
	@go tool cover -html=coverage.out

fmt:
	@go fmt ./...

vet:
	@go vet ./...

bench:
	@go test ./... -v -bench=.

clean:
	@rm -rf datadir
	@go clean
cleandata:
    rm -rf datadir/addrbook
    rm -rf datadir/blockchain.db
    rm -rf datadir/mavltree
    rm -rf chain33.log
