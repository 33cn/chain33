#!/bin/bash
set -x

DEP_DIR=$(dirname $(readlink -f "$0"))
cd $DEP_DIR
tar -xf gmssl_v2.0.tar.gz
cd gmssl_v2.0
./config
make
sudo make install
