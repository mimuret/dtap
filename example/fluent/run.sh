#!/bin/sh


docker-compose up -d 

curl http://localhost:9200 > /dev/null 2>&1
while [ $? -ne 0 ] 
do
  curl http://localhost:9200 > /dev/null 2>&1
  sleep 5
done

curl http://localhost:9200/_template/dtap -H "Content-Type: application/json" -XPUT -d '@../../misc/template.json' -v

dig @localhost github.com

