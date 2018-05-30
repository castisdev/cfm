#!/bin/sh

HOST=127.0.0.1:8080
#curl ${HOST}/tasks/copy
curl -X GET ${HOST}/tasks | jq .

