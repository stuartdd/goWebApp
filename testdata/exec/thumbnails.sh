#!/bin/bash

echo "goThumbnailTool" >> thumbnailsError.json
./goThumbnailTool >thumbnailsError.json 2>&1
if [ $? -eq 1 ]; then
  echo "goThumbnailTool Failed"
  exit 1
fi
