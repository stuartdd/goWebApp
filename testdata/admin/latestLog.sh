#!/bin/bash

path=$(pwd)/logs

file=$(ls $path -tp | grep -v /$ | head -1)

echo "Latest Log file: $file"
echo "-------------------------------------------------------------------------------------------------"
cat $path/$file
