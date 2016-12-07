#!/bin/sh

#####################
# resolve links - $0 may be a softlink
#####################

PRG="$0"
while [ -h "$PRG" ] ; do
  ls=`ls -ld "$PRG"`
  link=`expr "$ls" : '.*-> \(.*\)$'`
  if expr "$link" : '/.*' > /dev/null; then
    PRG="$link"
  else
    PRG=`dirname "$PRG"`/"$link"
  fi
done
PRGDIR=`dirname "$PRG"`

#####################
# start
#####################

PNAME=remote-tail

ifrun=$(ps -ef | grep $PNAME | grep -v grep)
if [ "$ifrun" != "" ];then
    echo "$PNAME is running..."
else
    echo "$PNAME was stopped."
    echo "$PNAME is starting..."
    nohup sh "$PRGDIR"/start.sh >/dev/null 2>&1 &
    echo "$PNAME was started."
fi