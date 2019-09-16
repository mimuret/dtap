#!/bin/bash -xe

docker-compose down
docker-compose up -d elasticsearch kafka-cluster

echo "add elasticsearch schema"
set +e
curl http://localhost:9200 > /dev/null 2>&1
while [ $? -ne 0 ] 
do
  sleep 5
  curl http://localhost:9200 > /dev/null 2>&1
done
set -e

echo "start kafka-cluster"
docker-compose up -d kafka-cluster

set +e
curl http://localhost:8081 > /dev/null 2>&1
while [ $? -ne 0 ] 
do
  sleep 5
  curl http://localhost:8081 > /dev/null 2>&1
done
set -e

#echo "regist schema"
#schema=$(jq -c . ../../assets/flat.avsc | sed 's/"/\\"/g')
#curl -vs --stderr - -XPOST -i -H "Content-Type: application/vnd.schemaregistry.v1+json" \
#--data "{\"schema\":\"$schema\"}" \
#http://localhost:8081/subject/query/versions

set +e
curl http://localhost:8083 > /dev/null 2>&1
while [ $? -ne 0 ] 
do
  sleep 5
  curl http://localhost:8083 > /dev/null 2>&1
done
set -e

echo "create connect"
curl -X PUT \
  http://localhost:8083/connectors/ElasticsearchSinkConnector/config \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  -d '{
  "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
  "type.name": "dnstap",
  "topics": "query",
  "tasks.max": "1",
  "key.ignore": "true",
  "schema.ignore": "true",
  "connection.url": "http://elasticsearch:9200/",
  "key.converter": "org.apache.kafka.connect.storage.StringConverter",
  "value.converter": "io.confluent.connect.avro.AvroConverter",
  "value.converter.schema.registry.url": "http://localhost:8081"
}'

echo "start other"
docker-compose up -d 
