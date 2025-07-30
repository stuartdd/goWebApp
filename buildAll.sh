#!/bin/bash
echo "*******************"
echo "* Build WebApp"
env | grep WebSer

if [ x"${WebServerRoot}" == "x" ]; then 
  echo "Value 'WebServerRoot' is not assigned to a variable"
	exit 1
fi

echo "*******************"

homeDir=$(pwd)
dpDir=$WebServerRoot

echo "Deploy to $WebServerRoot"
if [ -e $WebServerRoot/exec ]; then
  echo Clean
  rm -rf $WebServerRoot
fi

mkdir $WebServerRoot
cd $WebServerRoot

echo "COPY exec to $WebServerRoot"
cp -r ../testdata/exec .
if [ $? -eq 1 ]; then
  echo "Copy webtools Failed"
  exit 1
fi

echo "COPY goWebApp.json to $WebServerRoot"
cp ../goWebApp.json .
if [ $? -eq 1 ]; then
  echo "Copy goWebApp.json Failed"
  exit 1
fi

echo "COPY helptext.md to $WebServerRoot"
cp ../helptext.md .
if [ $? -eq 1 ]; then
  echo "Copy helptext.md Failed"
  exit 1
fi

cd $homeDir
echo "Build goWebApp to $WebServerRoot/goWebApp"
go build  -o $WebServerRoot/goWebApp goWebApp.go
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi

echo "Build webtools to $WebServerRoot/exec/webtools"
cd $homeDir/external
go build -o $WebServerRoot/exec/webtools 
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi

echo "Enable exec *.sh in $WebServerRoot/exec"
cd $WebServerRoot/exec
chmod +x *.sh

echo "*******************"
