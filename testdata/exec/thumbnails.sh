#!/bin/bash

./goThumbnailTool configThumbnail.json > thumbnails.txt 2>thumbnailsError.txt
if [ $? -eq 1 ]; then
  echo ": goThumbnailTool Failed" >> thumbnailsError.txt
  exit 1
fi

