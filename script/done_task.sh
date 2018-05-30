#!/bin/sh

if [ $# -ne 1 ];then
	echo "Usage: $0 <task id>"
	exit 1
fi

HOST=127.0.0.1:8080

curl -i -X PATCH -H 'Content-Type:application/json' ${HOST}/tasks/$1 -d'{"status": "done"}'
