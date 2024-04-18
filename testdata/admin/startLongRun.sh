#!/bin/bash
echo "./$1 $2 $3 &" >> lr.txt 
./$1 $2 $3 &
echo disown