#!/bin/sh
for((i=0;i<10;i++))
	do
	for((j=0;j<100;j++))
		do
		./send bty transfer -a 0.01 -n ceshi -t 18zB3xiWVnMwJa1GBkAqK9dnmRrPWANj81 -k 0XC4553726D9D04B6A58FF99BA4B4AEB47055F97F04514D30535CDE686365C2AF2
		sleep 0.1
		done
	sleep 2
	./send bty transfer -a 1 -n ceshi -t 1C5oJ8VbfQHjeAkstHVU3xA7yEerKeFacr -k 0X8FA41C1CA8BCB06E1455DEA656D2C2DE954D17FF18FE5E7A045115A0F739B0EF
	sleep 5
	done