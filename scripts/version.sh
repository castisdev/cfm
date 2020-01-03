#!/usr/bin/env bash

if [ -n "$1" ]; then
	Dir=$1
else
	Dir=$PWD
fi

LEAFDIR=`basename $Dir`
AppName=`grep -m 1 AppName $Dir/main.go |cut -s -d\" -f2`
CapitalAppName=`echo $AppName | tr '[a-z]' '[A-Z]'`
AppVersion=`grep -m 1 AppVersion $Dir/main.go |cut -s -d\" -f2`
AppPreRelVer=`grep -m 1 AppPreRelVer $Dir/main.go |cut -s -d\" -f2`
GoVer=`go version | cut -s -d' ' -f3`

if [ -n "$AppName" ]
then
	echo -n
else
	AppName=$LEAFDIR
fi

echo Dir = $Dir
echo AppName = $AppName
echo CapitalAppName = $CapitalAppName
echo AppVersion = $AppVersion
echo AppPreRelVer = $AppPreRelVer
echo GoVer = $GoVer
