#!/bin/bash

cd config
echo "--------------------------------------- $PWD"
go test

cd ../controllers
echo "--------------------------------------- $PWD"
go test

cd ../runCommand
echo "--------------------------------------- $PWD"
go test

cd ../server
echo "--------------------------------------- $PWD"
go test

cd ../tools
echo "--------------------------------------- $PWD"
go test

cd ..
echo "--------------------------------------- $PWD"
go test
