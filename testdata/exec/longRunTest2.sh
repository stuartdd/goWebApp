#!/bin/bash
rm -f LongRunTest2.txt
while true 
do
    echo "$(date '+%F %T') : LongRunTest2" >>  LongRunTest2.txt
    sleep 1
done
