#!/bin/bash

rm ../../originals/stuart/dirScanData.json

\time --format="%E" ./goWebApp config=tgo.json scan stuart
\time --format="%E" ./goWebApp config=tgo.json scan stuart
\time --format="%E" ./goWebApp config=tgo.json scan stuart
\time --format="%E" ./goWebApp config=tgo.json scan stuart

echo "*********************************** ALL PASS ****************************************"

cd ex   