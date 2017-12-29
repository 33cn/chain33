#!/bin/sh
for((i=0;i<1000000;i++))
do
./cli sendtoaddress 1H82nSA4WCbRhG8mMuWEoqSPC1WiKj3BWd 1BnRfFVcVcZCsZGykh5kgGQyJtNjVVYFai 0.0001 ""
echo "."
done
