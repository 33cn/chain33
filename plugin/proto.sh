#!/bin/sh

myPath="$1/proto"
if [ -a "$myPath" ]; then
    cd $myPath
    echo "make $1 ..."
    make
fi
