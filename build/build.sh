#!/bin/bash
rm -fr *.log
rm -fr logs
rm chain33
rm chain33-cli
export GOPATH=/home/litian/godev/
make -f Makefile.test
cp test.chain33.toml chain33.toml
