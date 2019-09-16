#!/bin/bash -x


docker-compose up -d 

curl http://localhost:9200 > /dev/null 2>&1
while [ $? -ne 0 ] 
do
  sleep 5
  curl http://localhost:9200 > /dev/null 2>&1
done

curl http://localhost:9200/_template/dtap -H "Content-Type: application/json" -XPUT -d '@../../misc/template.json' -v

dig @localhost github.com

curl http://localhost:9200/query-*
