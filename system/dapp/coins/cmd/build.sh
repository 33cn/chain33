#!/bin/sh

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && exit && pwd)"
OUT_DIR=$1
echo "-----only for test begin-----"
echo $DIR
echo $OUT_DIR
echo "-----only for test end-----"