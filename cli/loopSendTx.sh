#!/bin/sh
for((i=0;i<10000;i++))
do
./cli sendtoaddress 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt 1KRCM4wCLNmDyYPsjxntvqCEdARAEVT5T5 1 ceshi
echo "."
sleep 5
done
