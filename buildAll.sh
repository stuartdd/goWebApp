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
deployDir=$WebServerRoot

echo "Deploy to $deployDir"
if [ -e $deployDir/exec ]; then
  echo Clean
  rm -rf $deployDir
fi

mkdir $deployDir
cd $deployDir

echo "COPY exec to $deployDir"
cp -r ../testdata/exec .
if [ $? -eq 1 ]; then
  echo "Copy exec directory Failed"
  exit 1
fi

echo "COPY goWebApp.json to $deployDir"
cp ../goWebApp.json .
if [ $? -eq 1 ]; then
  echo "Copy goWebApp.json Failed"
  exit 1
fi

echo "COPY helptext.md to $deployDir"
cp ../helptext.md .
if [ $? -eq 1 ]; then
  echo "Copy helptext.md Failed"
  exit 1
fi

cd $homeDir
if [ "$arch" == "ARM" ]; then 
  echo "Build ARM goWebApp to $deployDir/goWebApp"
  env GOOS=linux GOARCH=arm go build  -o $deployDir/goWebApp goWebApp.go
  if [ $? -eq 1 ]; then
    echo "Build Failed: goWebApp ARM build failed"
    exit 1
  fi
else
  echo "Build INTEL goWebApp to $deployDir/goWebApp"
  go build  -o $deployDir/goWebApp goWebApp.go
  if [ $? -eq 1 ]; then
    echo "Build Failed: goWebApp INTEL build failed"
    exit 1
  fi
fi

echo "Enable exec *.sh in $deployDir/exec"
cd $deployDir/exec
chmod +x *.sh
if [ $? -eq 1 ]; then
  echo "Enable exec Failed"
  exit 1
fi

cd $homeDir/external

if [ "$arch" == "ARM" ]; then 
  echo "Build ARM webtools to $deployDir/exec/webtools"
  env GOOS=linux GOARCH=arm go build -o $deployDir/exec/webtools 
  if [ $? -eq 1 ]; then
    echo "Build Failed: webtools ARM build failed"
    exit 1
  fi
else
  echo "Build INTEL webtools to $deployDir/exec/webtools"
  go build -o $deployDir/exec/webtools 
  if [ $? -eq 1 ]; then
    echo "Build Failed: webtools INTEL build failed"
    exit 1
  fi
fi

cd $homeDir
if [ -e ../goThumbnailTool ]; then
    cd ../goThumbnailTool
    echo "Build Thumbnail tools '../goThumbnailTool'. Deploy to $arch $deployDir/exec"
    sh ./build.sh $arch $deployDir/exec/goThumbnailTool
    if [ $? -eq 1 ]; then
      echo "Build Failed: ../goThumbnailTool/build.sh returned an error"
      exit 1
    fi
    cp configThumbnail.json $deployDir/exec
    if [ $? -eq 1 ]; then
      echo "Build Failed: Failed to copy configThumbnail.json to $deployDir/exec"
      exit 1
    fi
    cd $homeDir
    cp $deployDir/exec/goThumbnailTool testdata/exec
    if [ $? -eq 1 ]; then
      echo "Build Failed: Failed to copy configThumbnail to ../testdata/exec"
      exit 1
    fi
    cp $deployDir/exec/configThumbnail.json testdata/exec
    if [ $? -eq 1 ]; then
      echo "Build Failed: Failed to copy configThumbnail.json to ../testdata/exec"
      exit 1
    fi

else
    echo "Build Failed: Thumbnail Nail tools project 'goThumbnailTool' does not exist in same dir as goWebApp!"
    exit 1
fi


echo "*******************"
