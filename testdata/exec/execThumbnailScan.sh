#!/bin/bash

dmesg | grep -i "$1" > execDmesg.raw

./webtools execDmesg.json

rm execDmesg.raw