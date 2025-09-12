#!/bin/bash

cd external
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd config
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd ../pictures
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

echo "***********************************  ALL PASS ****************************************"
cd ../server
echo "--------------------------------------- $PWD"
GOMAXPROCS=1 go test -run=XXX -bench=.
echo "*********************************** BENCHMARK ****************************************"
