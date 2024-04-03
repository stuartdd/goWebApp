#!/bin/bash
echo "--------------------------------------- $PWD"
go build -o testdata/admin/textToJson external/external.go 
if [ $? -eq 1 ]; then
  exit 1
fi


