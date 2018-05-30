#!/bin/sh

RAN_STR=$(env LC_CTYPE=C tr -dc "A-Z" < /dev/urandom | fold -w 7 | head -n1)
RAN_NUM=$(env LC_CTYPE=C tr -dc "0-9" < /dev/urandom | fold -w 14 | head -n1)
FILE_NAME=M${RAN_STR}SGL1500001_K${RAN_NUM}.mpg
HOST=http://localhost:8080



curl -i -X POST -H "Content-type:application/json" ${HOST}/tasks -d" \
{
    \"protocol\": \"netio\",
    \"src_ip\": \"172.16.8.101\",
    \"file_path\": \"/data2/${FILE_NAME}\"
}
"
