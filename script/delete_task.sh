#!/bin/sh

if [ $# -lt 1 ];then
	echo "Usage: $0 <task id>"
	exit 1
fi

for task_id in $@
do
	curl -i -X DELETE -H 'Content-Type:application/json' http://localhost:8080/tasks/$task_id
done
