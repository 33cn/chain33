#!/usr/bin/env bash
set -ex

# The script does automatic checking on a Go package and its sub-packages,
# including:
# 1. gofmt         (http://golang.org/cmd/gofmt/)
# 2. golint        (https://github.com/golang/lint)
# 3. go vet        (http://golang.org/cmd/vet)
# 4. gosimple      (https://github.com/dominikh/go-simple)
# 5. unconvert     (https://github.com/mdempsky/unconvert)
# 6. ineffassign   (https://github.com/gordonklaus/ineffassign)
# 7. race detector (http://blog.golang.org/race-detector)
# 8. test coverage (http://blog.golang.org/cover)

# gometalinter (github.com/alecthomas/gometalinter) is used to run each each
# static checker.

# To run on docker on windows, symlink /mnt/c to /c and then execute the script
# from the repo path under /c.  See:
# https://github.com/Microsoft/BashOnWindows/issues/1854
# for more details.

#Default GOVERSION
GOVERSION=${1:-1.10}
REPO=dcrd
DOCKER_IMAGE_TAG=decred-golang-builder-$GOVERSION

testrepo () {
  TMPFILE=$(mktemp)

  # Pull dependencies
  dep ensure
  if [ $? != 0 ]; then
    echo 'dep ensure failed'
    exit 1
  fi

  # Check linters
  gometalinter --vendor --disable-all --deadline=10m \
    --enable=gofmt \
    --enable=golint \
    --enable=vet \
    --enable=gosimple \
    --enable=unconvert \
    --enable=ineffassign \
    ./...
  if [ $? != 0 ]; then
    echo 'gometalinter has some complaints'
    exit 1
  fi

  # Check tests
  env GORACE='halt_on_error=1' go test -race ./...
  if [ $? != 0 ]; then
    echo 'go tests failed'
    exit 1
  fi

  echo "------------------------------------------"
  echo "Tests completed successfully!"
}

if [ $GOVERSION == "local" ]; then
    testrepo
    exit
fi

mkdir -p ~/.cache

if [ -f ~/.cache/$DOCKER_IMAGE_TAG.tar ]; then
	# load via cache
	docker load -i ~/.cache/$DOCKER_IMAGE_TAG.tar
	if [ $? != 0 ]; then
		echo 'docker load failed'
		exit 1
	fi
else
	# pull and save image to cache
	docker pull decred/$DOCKER_IMAGE_TAG
	if [ $? != 0 ]; then
		echo 'docker pull failed'
		exit 1
	fi
	docker save -o ~/.cache/$DOCKER_IMAGE_TAG.tar decred/$DOCKER_IMAGE_TAG
	if [ $? != 0 ]; then
		echo 'docker save failed'
		exit 1
	fi
fi

docker run --rm -it -v $(pwd):/src decred/$DOCKER_IMAGE_TAG /bin/bash -c "\
  rsync -ra --filter=':- .gitignore'  \
  /src/ /go/src/github.com/decred/$REPO/ && \
  cd github.com/decred/$REPO/ && \
  bash run_tests.sh local"
if [ $? != 0 ]; then
	echo 'docker run failed'
	exit 1
fi
