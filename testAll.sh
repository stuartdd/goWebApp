#!/bin/bash
if [ ! -d testdata/logs ]; then
  mkdir -p testdata/logs
  if [ $? -gt 0 ]; then
    echo "Check Path: Could not create testdata/logs"
    exit 1
  fi
  echo "Check Path: Created testdata/logs"
fi

cd external
echo "--------------------------------------- $PWD"
go test
if [ $? -eq 1 ]; then
  exit 1
fi

cd ../config
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
