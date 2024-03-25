#!/bin/bash

cd config
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd ../controllers
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd ../runCommand
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd ../server
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd ../logging
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd ..
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

