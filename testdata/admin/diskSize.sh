#!/bin/bash
df -h | grep "/nvme" | ./textToJson dsConfig.json