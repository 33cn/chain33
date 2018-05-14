#!/bin/bash
echo -n
rm -fr *.log
rm -fr logs
make -f Makefile.test
PID=`ps -ef|grep 'dlv --headless=true'|grep -v grep|awk '{print $2}'`
kill -9 $PID
PID=`ps -ef|grep 'chain33 -f test.chain33.toml'|grep -v grep|awk '{print $2}'`
kill -9 $PID
# 根据自己情况，是否需要调试，使用下面两个命令
# ./chain33 -f test.chain33.toml
dlv --headless=true --listen=:2345 --api-version=2 exec chain33 -- -f test.chain33.toml
