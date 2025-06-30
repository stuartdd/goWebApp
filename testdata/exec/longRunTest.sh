#!/bin/bash
rm -f LongRunTest.txt
while true 
do
    echo "$(date '+%F %T') : LongRunTest" >>  LongRunTest.txt
    sleep 1
done
