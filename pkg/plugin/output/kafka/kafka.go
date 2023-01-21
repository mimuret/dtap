/*
 * Copyright (c) 2022 Manabu Sonoda
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

package kafka

import (
	_ "embed"
	"encoding/binary"
	"fmt"

	json "github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/riferrei/srclient"

	"github.com/dangkaka/go-kafka-avro"
	"github.com/linkedin/goavro"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/output"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"

	"github.com/Shopify/sarama"
)

//go:embed assets/flat.avsc
var valueSchemaStr string

//go:embed assets/key.avsc
var keySchemaStr string

func init() {
	_ = registry.RegisterOutputPlugin("kafka", Setup)
}

func Setup(bs json.RawMessage) (types.OutputPlugin, error) {
	var err error
	s := &Kafka{}
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, errors.Wrap(err, "failed to decode config")
	}

	s.samaraConfig = sarama.NewConfig()
	s.samaraConfig.Producer.Return.Successes = true
	s.samaraConfig.Producer.Return.Errors = true
	s.samaraConfig.Producer.Retry.Max = int(s.KafkaConfig.Retry)

	s.keyCodec, err = goavro.NewCodec(keySchemaStr)
	if err != nil {
		return nil, err
	}

	s.valueCodec, err = goavro.NewCodec(valueSchemaStr)
	if err != nil {
		return nil, err
	}
	s.DnstapOutput = output.NewDnstapOutput(s, s.MaxRetry)

	return s, nil
}

type KafkaClient interface {
	Add(string, string, []byte, []byte) error
}

type Kafka struct {
	plugin.PluginCommon
	KafkaConfig KafkaConfig
	*output.DnstapOutput

	samaraConfig  *sarama.Config
	producer      sarama.SyncProducer
	registry      *kafka.CachedSchemaRegistryClient
	valueCodec    *goavro.Codec
	valueSchemaID []byte
	keyCodec      *goavro.Codec
	keySchemaID   []byte
	oc            *types.OutputContext
}

func (f *Kafka) SetOutputContext(oc *types.OutputContext) {
	f.oc = oc
}

type OutputType string

var (
	OutputTypeJSON     OutputType = "json"
	OutputTypePtoroBuf OutputType = "protobuf"
	OutputTypeAvero    OutputType = "avero"
)

type KafkaConfig struct {
	Hosts            []string
	SchemaRegistries []string
	Retry            uint
	Topic            string
	Key              string
	OutputType       OutputType
}

func (o *Kafka) Open() error {
	var err error
	o.producer, err = sarama.NewSyncProducer(o.KafkaConfig.Hosts, o.samaraConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create kafka producer")
	}
	o.registry = kafka.NewCachedSchemaRegistryClient(o.KafkaConfig.SchemaRegistries)
	if o.KafkaConfig.OutputType == OutputTypeAvero {
		if o.valueSchemaID, err = o.getSchemaID(o.KafkaConfig.Topic+"-value", valueSchemaStr); err != nil {
			return errors.Wrap(err, "failed to get value schema id")
		}
		if o.keySchemaID, err = o.getSchemaID(o.KafkaConfig.Topic+"-key", keySchemaStr); err != nil {
			return errors.Wrap(err, "failed to get key schema id")
		}
	}
	return nil
}

func (o *Kafka) getSchemaID(subject string, schemaStr string) ([]byte, error) {
	var (
		err    error
		schema *srclient.Schema
	)
	for _, host := range o.KafkaConfig.SchemaRegistries {
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

func (o *Kafka) GetEncoder(v interface{}, codec *goavro.Codec, schemaID []byte) (sarama.Encoder, error) {
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

func (o *Kafka) Write(dm *types.DnstapMessage) error {
	var err error
	var v, k sarama.Encoder
	switch o.KafkaConfig.OutputType {
	case OutputTypePtoroBuf:
		k = sarama.ByteEncoder(o.KafkaConfig.Key)
		v = sarama.ByteEncoder(dm.GetRaw())
	case OutputTypeAvero:
		mapString, err := dm.ConvertV1Flat()
		if err != nil {
			return err
		}
		if v, err = o.GetEncoder(mapString, o.valueCodec, o.valueSchemaID); err != nil {
			return err
		}
		if k, err = o.GetEncoder(o.KafkaConfig.Key, o.keyCodec, o.keySchemaID); err != nil {
			return err
		}
	case OutputTypeJSON:
		buf, err := dm.ConvertV1JSON()
		if err != nil {
			return err
		}
		k = sarama.StringEncoder(o.KafkaConfig.Key)
		v = sarama.StringEncoder(buf)
	}

	msg := &sarama.ProducerMessage{
		Topic: o.KafkaConfig.Topic,
		Key:   k,
		Value: v,
	}
	_, _, err = o.producer.SendMessage(msg)

	return err
}

func (o *Kafka) Close() {
	o.producer.Close()
}
