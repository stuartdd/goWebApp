#!/bin/bash
echo "--------------------------------------- Build goWebApp"
go build -o goWebApp goWebApp.go 
if [ $? -eq 1 ]; then
  exit 1
fi

echo "--------------------------------------- Build textToJson"
go build -o testdata/admin/textToJson external/external.go 
if [ $? -eq 1 ]; then
  exit 1
fi


