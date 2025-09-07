#!/bin/bash
rm -f LongRunTest1.txt
rm -f LongRunTest1Error.txt
echo "LongRunTest1 is a test exec" > LongRunTest1Error.txt
while true 
do
    echo "$(date '+%F %T') : LongRunTest1" >>  LongRunTest1.txt
    sleep 1
done
