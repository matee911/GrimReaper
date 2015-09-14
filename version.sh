#!/bin/sh

CHANGED=`git diff-index --name-only HEAD --`
if [ -n "$CHANGED" ]
then
	VERSION="`git describe --tag --always`-uncommited (Built `date -u +%Y-%m-%d\ %H:%M:%S` by `whoami`@`hostname`)"
	echo $VERSION
	exit 0
fi

VERSION=`git describe --tag --exact-match`
RET=$?

if [ $RET != 0 ]
then
	VERSION="`git describe --tag --always`-dev (Built `date -u +%Y-%m-%d\ %H:%M:%S` by `whoami`@`hostname`)"
fi

echo $VERSION
