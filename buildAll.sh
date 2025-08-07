#!/bin/bash
echo "*******************"
if [ "$1" == "ARM" ]; then 
  echo "* Build WebApp $1"
  arch=ARM
else
  echo "* Build WebApp INTEL (Default)"
  arch=INTEL
fi

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
  echo "Copy exec directory Failed"
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
if [ "$arch" == "ARM" ]; then 
  echo "Build ARM goWebApp to $WebServerRoot/goWebApp"
  env GOOS=linux GOARCH=arm go build  -o $WebServerRoot/goWebApp goWebApp.go
  if [ $? -eq 1 ]; then
    echo "Build Failed: goWebApp ARM build failed"
    exit 1
  fi
else
  echo "Build INTEL goWebApp to $WebServerRoot/goWebApp"
  go build  -o $WebServerRoot/goWebApp goWebApp.go
  if [ $? -eq 1 ]; then
    echo "Build Failed: goWebApp INTEL build failed"
    exit 1
  fi
fi

echo "Enable exec *.sh in $WebServerRoot/exec"
cd $WebServerRoot/exec
chmod +x *.sh
if [ $? -eq 1 ]; then
  echo "Enable exec Failed"
  exit 1
fi

cd $homeDir/external

if [ "$arch" == "ARM" ]; then 
  echo "Build ARM webtools to $WebServerRoot/exec/webtools"
  env GOOS=linux GOARCH=arm go build -o $WebServerRoot/exec/webtools 
  if [ $? -eq 1 ]; then
    echo "Build Failed: webtools ARM build failed"
    exit 1
  fi
else
  echo "Build INTEL webtools to $WebServerRoot/exec/webtools"
  go build -o $WebServerRoot/exec/webtools 
  if [ $? -eq 1 ]; then
    echo "Build Failed: webtools INTEL build failed"
    exit 1
  fi
fi

cd $homeDir
if [ -e ../goThumbnailTool ]; then
    cd ../goThumbnailTool
    echo "Build Thumbnail tools '../goThumbnailTool'. Deploy to $arch $WebServerRoot/exec"
    sh ./build.sh $arch $WebServerRoot/exec/goThumbnailTool
    if [ $? -eq 1 ]; then
      echo "Build Failed: ../goThumbnailTool/build.sh returned an error"
      exit 1
    fi
    cp configThumbnail.json $WebServerRoot/exec
    if [ $? -eq 1 ]; then
      echo "Build Failed: Failed to copy configThumbnail.json to $WebServerRoot/exec"
      exit 1
    fi
    cd $homeDir
    cp $WebServerRoot/exec/goThumbnailTool testdata/exec
    if [ $? -eq 1 ]; then
      echo "Build Failed: Failed to copy configThumbnail to ../testdata/exec"
      exit 1
    fi
    cp $WebServerRoot/exec/configThumbnail.json testdata/exec
    if [ $? -eq 1 ]; then
      echo "Build Failed: Failed to copy configThumbnail.json to ../testdata/exec"
      exit 1
    fi

else
    echo "Build Failed: Thumbnail Nail tools project 'goThumbnailTool' does not exist in same dir as goWebApp!"
    exit 1
fi


echo "*******************"
