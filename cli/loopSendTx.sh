#!/bin/sh
for((i=0;i<10000;i++))
do
./cli sendtoaddress 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv 1KRCM4wCLNmDyYPsjxntvqCEdARAEVT5T5 0.001 ceshi
echo "."
sleep 0.1
done
