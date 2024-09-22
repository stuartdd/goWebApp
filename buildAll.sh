#!/bin/bash
echo "--------------------------------------- Build goWebApp"
go build -o goWebApp goWebApp.go 
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi

echo "--------------------------------------- Build webtools"
cd external
go build -o ../testdata/admin/webtools 
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi


