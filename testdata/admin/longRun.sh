#!/bin/bash

LOGFILE=logs/longRun-`date +%Y-%m-%d`.log
while true 
do
    echo "$(date '+%F %T') : Task" >> $LOGFILE
    sleep 1
done