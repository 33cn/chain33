#!/bin/sh
for((i=0;i<10000;i++))
do
./cli sendtoaddress 15kqZpgW8qiyRsoWmGHS3r1Vc6avFermQh 1KRCM4wCLNmDyYPsjxntvqCEdARAEVT5T5 0.001 ceshi
echo "."
sleep 0.1
done
