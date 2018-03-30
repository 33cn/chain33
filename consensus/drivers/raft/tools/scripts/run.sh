#!/usr/bin/env bash
#Program:
# This is a chain33 deploy scripts!
if [ $1 == "start" ]
then
  nohup ./chain33 &
  echo $! >chain33.pid
elif [ $1 == "stop" ]
then
  PID=$(cat chain33.pid)
  kill -9 $PID
  rm -rf  chain33.pid
elif [ $1 == "clear" ]
then
  rm -rf nohup.out
  rm -rf chain33.log
  rm -rf grpc33.log
  rm -rf chain33_raft*
  rm -rf datadir/
else
  echo "Usage: ./run.sh [start,stop,clear]"
fi
