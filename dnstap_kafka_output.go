/*
 * Copyright (c) 2019 Manabu Sonoda
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dtap

import (
	_ "embed"
	"encoding/binary"
	"fmt"

	json "github.com/goccy/go-json"
	"github.com/riferrei/srclient"

	"github.com/Shopify/sarama"
	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/golang/protobuf/proto"
	"github.com/linkedin/goavro"
)

//go:embed assets/flat.avsc
var valueSchemaStr string

//go:embed assets/key.avsc
var keySchemaStr string

type KafkaClient interface {
	Add(string, string, []byte, []byte) error
}

type DnstapKafkaOutput struct {
	config        *OutputKafkaConfig
	kafkaConfig   *sarama.Config
	producer      sarama.SyncProducer
	valueSchemaID []byte
	valueCodec    *goavro.Codec
	keySchemaID   []byte
	keyCodec      *goavro.Codec
}

func NewDnstapKafkaOutput(config *OutputKafkaConfig, params *DnstapOutputParams) (*DnstapOutput, error) {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Retry.Max = int(config.GetRetry())

	keyCodec, err := goavro.NewCodec(keySchemaStr)
	if err != nil {
		return nil, err
	}

	valueCodec, err := goavro.NewCodec(valueSchemaStr)
	if err != nil {
		return nil, err
	}

	params.Handler = &DnstapKafkaOutput{
		config:      config,
		kafkaConfig: kafkaConfig,
		keyCodec:    keyCodec,
		valueCodec:  valueCodec,
	}

	return NewDnstapOutput(params), nil
}

func (o *DnstapKafkaOutput) open() error {
	var err error
	o.producer, err = sarama.NewSyncProducer(o.config.Hosts, o.kafkaConfig)
	if err != nil {
		return fmt.Errorf("failed to create kafka producer: %w", err)
	}
	if o.config.GetOutputType() == "avro" {
		if o.valueSchemaID, err = o.getSchemaID(o.config.GetTopic()+"-value", valueSchemaStr); err != nil {
			return fmt.Errorf("failed to get schema id: %w", err)
		}
		if o.keySchemaID, err = o.getSchemaID(o.config.GetTopic()+"-key", keySchemaStr); err != nil {
			return fmt.Errorf("failed to get schema id: %w", err)
		}
	}
	return nil
}
func (o *DnstapKafkaOutput) getSchemaID(subject string, schemaStr string) ([]byte, error) {
	var (
		err    error
		schema *srclient.Schema
	)
	for _, host := range o.config.GetSchemaRegistries() {
		client := srclient.CreateSchemaRegistryClient(host)
		schema, err = client.GetLatestSchema(subject)
		if err != nil {
			continue
		}
		if schema == nil {
			schema, err = client.CreateSchema(subject, schemaStr, srclient.Avro)
			if err != nil {
				panic(fmt.Sprintf("Error creating the schema %s", err))
			}
		}
	}
	if err != nil {
		return nil, err
	}
	val := make([]byte, 4)
	binary.BigEndian.PutUint32(val, uint32(schema.ID()))
	return val, nil
}

func (o *DnstapKafkaOutput) GetEncoder(v interface{}, codec *goavro.Codec, schemaID []byte) (sarama.Encoder, error) {
	binary, err := codec.BinaryFromNative(nil, v)
	if err != nil {
		return nil, err
	}
	var binaryMsg []byte
	// first byte is magic byte, always 0 for now
	binaryMsg = append(binaryMsg, byte(0))
	//4-byte schema ID as returned by the Schema Registry
	binaryMsg = append(binaryMsg, schemaID...)
	//avro serialized data in Avro’s binary encoding
	binaryMsg = append(binaryMsg, binary...)

	return sarama.ByteEncoder(binaryMsg), nil
}

func (o *DnstapKafkaOutput) write(frame []byte) error {
	var v, k sarama.Encoder
	if o.config.GetOutputType() == "protobuf" {
		k = sarama.ByteEncoder(o.config.GetKey())
		v = sarama.ByteEncoder(frame)
	} else {
		dt := dnstap.Dnstap{}
		if err := proto.Unmarshal(frame, &dt); err != nil {
			return err
		}
		data, err := FlatDnstap(&dt, &o.config.Flat)
		if err != nil {
			return err
		}
		if o.config.GetOutputType() == "avro" {
			var err error
			mapString := data.ToMapString()
			if v, err = o.GetEncoder(mapString, o.valueCodec, o.valueSchemaID); err != nil {
				return err
			}
			if k, err = o.GetEncoder(o.config.GetKey(), o.keyCodec, o.keySchemaID); err != nil {
				return err
			}
		} else {
			buf, err := json.Marshal(data)
			if err != nil {
				return err
			}
			k = sarama.StringEncoder(o.config.GetKey())
			v = sarama.StringEncoder(buf)
		}
	}

	msg := &sarama.ProducerMessage{
		Topic: o.config.GetTopic(),
		Key:   k,
		Value: v,
	}
	_, _, err := o.producer.SendMessage(msg)

	return err
}

func (o *DnstapKafkaOutput) close() {
}
