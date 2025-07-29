#!/bin/bash
rm -f LongRunTest1.txt
while true 
do
    echo "$(date '+%F %T') : LongRunTest1" >>  LongRunTest1.txt
    sleep 1
done
