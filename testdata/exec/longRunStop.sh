#!/bin/bash
kill $1
echo "{\"msg\":\"KILL\",\"pid\":\"$1\"}"
