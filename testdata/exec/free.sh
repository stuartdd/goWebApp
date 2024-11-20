#!/bin/bash

free | grep -i "$1" > freeOut.raw

./webtools freeConfig.json

rm freeOut.raw