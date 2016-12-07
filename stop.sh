#!/bin/bash


aid=`ps aux | grep remote-tail | grep -v grep |awk '{print $2}'`

kill -9 $aid