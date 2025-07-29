#!/bin/bash
homeDir=$(pwd)
echo "--------------------------------------- Build goWebApp"
go build -o goWebApp goWebApp.go 
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi

echo "--------------------------------------- Build webtools"
cd external
go build -o ../testdata/exec/webtools 
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi

cd $homeDir
if ! test -f goWebApp; then
  echo "Build Deploy Failed. Cannot find goWebApp"
  exit 1
fi

dpDir=.deploy
echo "--------------------------------------- Build Deploy $dpDir"
if [ -e $dpDir ]; then
  echo Clean
  rm -rf $dpDir
fi

mkdir $dpDir
cd $dpDir

echo "COPY goWebApp to $dpDir/goWebApp"
cp ../goWebApp .
if [ $? -eq 1 ]; then
  echo "Copy goWebApp Failed"
  exit 1
fi

echo "COPY run to $dpDir/goWebApp"
cp ../run .
if [ $? -eq 1 ]; then
  echo "Copy run Failed"
  exit 1
fi

echo "COPY exec to $dpDir/goWebApp/exec"
cp -r ../testdata/exec .
if [ $? -eq 1 ]; then
  echo "Copy webtools Failed"
  exit 1
fi

echo "COPY goWebApp.json to $dpDir/goWebApp"
cp ../goWebApp.json .
if [ $? -eq 1 ]; then
  echo "Copy goWebApp.json Failed"
  exit 1
fi

echo "COPY helptext.md to $dpDir/goWebApp"
cp ../helptext.md .
if [ $? -eq 1 ]; then
  echo "Copy helptext.md Failed"
  exit 1
fi
