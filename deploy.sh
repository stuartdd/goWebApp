#!/bin/bash

./buildAll.sh ARM
if [ $? -eq 1 ]; then
  echo "Deploy:BuildAll failed"
  exit 1
fi

DEPLOY=$WebServerRoot
if [ ! -e $DEPLOY ]; then
  echo "Deploy dir '$DEPLOY' has not been created"
  exit 1
fi

rsync -avz -e 'ssh' $DEPLOY/ pi@192.168.1.98:/home/pi/server
