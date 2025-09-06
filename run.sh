#!/bin/bash

echo "########################################################"
echo "Checking environment vars:"
env | grep WebSer
echo "########################################################"

if [ x"${WebServerParent}" == "x" ]; then 
  echo "Value 'WebServerParent' is not assigned to a variable"
  echo "Should be dir that contains WebServerRoot --> ${WebServerRoot}"
  exit 1
fi

if [ x"${WebServerRoot}" == "x" ]; then 
  echo "Value 'WebServerRoot' is not assigned to a variable"
  echo "Should be dir that executable is in. The dir the server is deployed to"
  exit 1
fi

if [ x"${WebServerData}" == "x" ]; then 
  echo "Value 'WebServerData' is not assigned to a variable"
  echo "Should be dir outside of "WebServerRoot" that contains volatile user data"
	exit 1
fi

if [ x"${WebServerHome}" == "x" ]; then 
  echo "Value 'WebServerHome' is not assigned to a variable"
  echo "Should be dir the web files are dployed to. Should contain 'static' dir. E.g. web/static"
	exit 1
fi

if [ x"${WebServerPictures}" == "x" ]; then 
  echo "Value 'WebServerPictures' is not assigned to a variable"
  echo "Should be the root of the users pictures dir. Used by location 'originals" in user data in UserDataPath
  exit 1
fi

if [ x"${WebServerThumbnails}" == "x" ]; then 
  echo "Value 'WebServerThumbnails' is not assigned to a variable"
  echo "Should be the root of the thumbnails dir. Used by location 'thumbs' in user data in UserDataPath"
  exit 1
fi


echo "Checking Paths exist:"
if [ ! -d $WebServerParent ]; then
  echo "Deploy dir '$WebServerParent' does not exist"
  exit 1
fi

if [ ! -d $WebServerRoot ]; then
  mkdir -p $WebServerRoot
    if [ $? -gt 0 ]; then
    echo "Check Path: Could not create $WebServerRoot"
    exit 1
  fi
fi

if [ ! -d $WebServerRoot/exec ]; then
  mkdir -p $WebServerRoot/exec
    if [ $? -gt 0 ]; then
    echo "Check Path: Could not create $WebServerRoot/exec"
    exit 1
  fi
fi

if [ ! -d $WebServerData ]; then
  echo "Check Path: '$WebServerData' does not exist"
  exit 1
fi

if [ ! -d $WebServerHome ]; then
  echo "Check Path:  '$WebServerHome' does not exist. Web server not deployed!"
  exit 1
fi

if [ ! -d $WebServerHome/static ]; then
  echo "Check Path:  '$WebServerHome/static' does not exist. Web server not deployed!"
  exit 1
fi

if [ ! -d $WebServerPictures ]; then
  echo "Deploy dir '$WebServerPictures' does not exist"
  exit 1
fi

if [ ! -d $WebServerThumbnails ]; then
  echo "Deploy dir '$WebServerThumbnails' does not exist"
  exit 1
fi

if [ "$1" == "test" ]; then 
  echo "Exec tests completed OK"
  exit 0
fi

cd $WebServerRoot
echo "########################################################"
echo "Running in: $WebServerRoot"

if [ ! -e $WebServerRoot/goWebApp ]; then
  echo "Exec file '$WebServerRoot/goWebApp' does not exist"
  exit 1
fi

if [ ! -e $WebServerRoot/exec/webtools ]; then
  echo "Exec file '$WebServerRoot/exec/webtools' does not exist"
  exit 1
fi


 

while true
do
  ./goWebApp config=goWebApp.json -vr
  RESP=$?
  echo "########################################################"
  echo "Response: $RESP"
  if [ $RESP -eq 11 ]; then
    echo "Server stopped"
    exit 1
  fi
  if [ $RESP -eq 10 ]; then
    echo "Server already running"
    exit 1
  fi
  if [ $RESP -eq 1 ]; then
    echo "Server error"
    exit 1
  fi
  if [ $RESP -eq 2 ]; then
    echo "Server error"
    exit 1
  fi
  if [ $RESP -gt 100 ]; then
    echo "Server error"
    exit 1
  fi
  echo "Server restarted"
done

