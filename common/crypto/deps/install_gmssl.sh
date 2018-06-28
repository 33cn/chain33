#!/bin/bash
set -x

DEP_DIR=$PWD
cd "$DEP_DIR" || exit 1
tar -xf gmssl_v2.0.tar.gz
cd gmssl_v2.0 || exit 1
./config
make
sudo make install
