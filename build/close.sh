#!/bin/bash
PID=`ps -ef|grep 'dlv --headless=true'|grep -v grep|awk '{print $2}'`
kill -9 $PID
PID=`ps -ef|grep 'chain33 -f test.chain33.toml'|grep -v grep|awk '{print $2}'`
kill -9 $PID
