#!/bin/bash

dmesg | grep -i "$1" > execDmesg.raw

./textToJson execDmesg.json

rm execDmesg.raw