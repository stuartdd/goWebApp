#!/bin/bash
./run.sh test
if [ $? -eq 1 ]; then
  echo "Run test Failed"
  exit 1
fi

echo "*******************"
if [ "$1" == "ARM" ]; then 
  echo "* Build WebApp For ARM (RaspberryPi)"
  arch=ARM
else
  echo "* Build WebApp For INTEL (Default)"
  arch=INTEL
fi

if [ x"${WebServerRoot}" == "x" ]; then 
  echo "Value 'WebServerRoot' is not assigned to a variable"
  echo "This is where the files will be packaged for remote or local deployment"
  exit 1
fi

echo "*******************"

homeDir=$(pwd)
deployDir=$WebServerRoot

echo "Deploy to $deployDir"
if [ -e $deployDir ]; then
  echo "Clean Deployment root: $deployDir"
  rm -rf $deployDir
fi

mkdir $deployDir
if [ $? -eq 1 ]; then
  echo "Failed to create Deployment root: $deployDir"
  exit 1
fi

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

echo "COPY run to $deployDir"
cp ../run.sh .
if [ $? -eq 1 ]; then
  echo "Copy run.sh Failed"
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
  
  cp $deployDir/exec/webtools $homeDir/testdata/exec/webtools
  if [ $? -eq 1 ]; then
    echo "Build Failed: could not copy webtools to $homeDir/testdata/exec/webtools"
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
    if [ "$arch" == "INTEL" ]; then 
      cp $deployDir/exec/goThumbnailTool $homeDir/testdata/exec
      if [ $? -eq 1 ]; then
        echo "Build Failed: Failed to copy goThumbnailTool to .$homeDir/testdata/exec/exec/goThumbnailTool"
        exit 1
      fi
      cp $deployDir/exec/configThumbnail.json $homeDir/testdata/exec
      if [ $? -eq 1 ]; then
        echo "Build Failed: Failed to copy configThumbnail.json to $homeDir/testdata/exec"
        exit 1
      fi
    fi
else
    echo "Build Failed: Thumbnail Nail tools project 'goThumbnailTool' does not exist in same dir as goWebApp!"
    exit 1
fi


echo "*******************"
