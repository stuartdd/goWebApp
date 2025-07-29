#!/bin/bash
homeDir=$(pwd)
dpDir=deployToPi

echo "--------------------------------------- Clean Deploy $dpDir"
if [ -e $dpDir ]; then
  echo Clean
  rm -rf $dpDir
fi

mkdir $dpDir

echo "--------------------------------------- Build goWebApp > $dpDir"
env GOOS=linux GOARCH=arm go build -o $dpDir/goWebApp -ldflags="-s -w" goWebApp.go 
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi

if ! test -f $dpDir/goWebApp; then
  echo "Build Deploy Failed. Cannot find goWebApp"
  exit 1
fi

echo "--------------------------------------- Build webtools"
cd external
env GOOS=linux GOARCH=arm go build -o ../testdata/exec/webtools  -ldflags="-s -w" . 
if [ $? -eq 1 ]; then
  echo "Build Failed"
  exit 1
fi

cd $homeDir
if ! test -f testdata/exec/webtools; then
  echo "Build Deploy Failed. Cannot find testdata/exec/webtools"
  exit 1
fi

cd $dpDir
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

cd $homeDir
echo "TRANSFER to server"
rsync -avz -e 'ssh' $dpDir pi@192.168.1.243:/home/pi/server/$dpDir
if [ $? -eq 1 ]; then
  echo "TRANSFER to server Failed"
  exit 1
fi