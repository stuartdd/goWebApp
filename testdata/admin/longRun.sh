#!/bin/bash

LOGFILE=$1
while true 
do
    echo "$(date '+%F %T') : Task" >> $LOGFILE
    sleep 1
done