#!/bin/sh
for((i=0;i<10;i++))
	do
	for((j=0;j<100;j++))
		do
		./send bty transfer -a 0.01 -n ceshi -t 1KGQ93eqs64966HdB2r5K5q6V1D58R32g1 -k 0X641446B566CF20A5614F7F6A183B119DBF965ACAD7FC56E765D27C6154B70F94
		sleep 0.1
		done
	sleep 2
	./send bty transfer -a 1 -n ceshi -t 1Jk74WJV2ahqDEhm9HLscTRzc6QWzzW5tf -k 0X9A393D4482633905696AEAF45FDD4223CFCABE0DF44541FD8693CA562E0FAC78
	sleep 5
	done