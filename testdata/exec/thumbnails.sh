#!/bin/bash
# rm -f tn.txt
./goThumbnailTool >tn.txt 2>&1
if [ $? -eq 1 ]; then
  echo "goThumbnailTool Failed"
  exit 1
fi
