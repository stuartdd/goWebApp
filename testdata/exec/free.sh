#!/bin/bash

free > freeOut.raw
uname -r >> freeOut.raw
uname -v >> freeOut.raw
uname -p >> freeOut.raw

./webtools freeConfig.json

rm freeOut.raw

exit 0
