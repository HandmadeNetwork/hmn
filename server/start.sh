#!/bin/bash

mkdir -p /home/hmn/log
nohup /home/hmn/hmn/hmn > /home/hmn/log/hmn.log 2>&1 &
echo $! > /home/hmn/hmn.pid
