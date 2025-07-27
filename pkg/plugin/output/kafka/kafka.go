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
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/riferrei/srclient"

	"github.com/dangkaka/go-kafka-avro"
	"github.com/linkedin/goavro"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/output"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"

	"github.com/Shopify/sarama"
)

const PLUGIN_NAME = "kafka"

//go:embed assets/flat.avsc
var valueSchemaStr string

//go:embed assets/key.avsc
var keySchemaStr string

func init() {
	_ = registry.RegisterOutputPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.OutputBlock) (types.OutputPlugin, error) {
	var err error
	s := &Kafka{
		OutputBlock:   *cfg,
		OutputFilters: &types.OutputFilters{},
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, s)
	if diags.HasErrors() {
		return nil, plugin.PluginError(s, "failed to setup otel-log plugin: %w", errors.Join(diags.Errs()...))
	}

	s.samaraConfig = sarama.NewConfig()
	s.samaraConfig.Producer.Return.Successes = true
	s.samaraConfig.Producer.Return.Errors = true
	s.samaraConfig.Producer.Retry.Max = int(s.Retry)

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

// The kafka plugin outputs messages to the kafka server.
type Kafka struct {
	config.OutputBlock

	// Hosts is the list of kafka hosts.
	Hosts []string `hcl:"hosts"`

	// SchemaRegistries is the list of schema registry hosts.
	SchemaRegistries []string `hcl:"schema_registries,optional"`

	// Retry is the number of retries to send the message.
	Retry uint `hcl:"retry,optional"`

	// Topic is the kafka topic to send the message.
	Topic string `hcl:"topic"`

	// Key is the key to send the message.
	Key string `hcl:"key,optional"`

	// OutputType is the type of output.
	OutputType OutputType `hcl:"output_type,optional"`

	// OutputFilters is the filters to apply to the output.
	OutputFilters *types.OutputFilters `hcl:"output_filters,block"`

	// MaxRetry is the maximum number of retries to connect to server.
	MaxRetry uint `hcl:"max_retry,optional"`

	// sarama config
	samaraConfig  *sarama.Config
	producer      sarama.SyncProducer
	registry      *kafka.CachedSchemaRegistryClient
	valueCodec    *goavro.Codec
	valueSchemaID []byte
	keyCodec      *goavro.Codec
	keySchemaID   []byte

	*output.DnstapOutput
}

type OutputType string

var (
	OutputTypeJSON     OutputType = "json"
	OutputTypePtoroBuf OutputType = "protobuf"
	OutputTypeAvero    OutputType = "avero"
)

// kafka config
type KafkaConfig struct {
}

func (o *Kafka) Open(context.Context) error {
	var err error
	o.producer, err = sarama.NewSyncProducer(o.Hosts, o.samaraConfig)
	if err != nil {
		return fmt.Errorf("failed to create kafka producer: %w", err)
	}
	o.registry = kafka.NewCachedSchemaRegistryClient(o.SchemaRegistries)
	if o.OutputType == OutputTypeAvero {
		if o.valueSchemaID, err = o.getSchemaID(o.Topic+"-value", valueSchemaStr); err != nil {
			return fmt.Errorf("failed to get value schema id: %w", err)
		}
		if o.keySchemaID, err = o.getSchemaID(o.Topic+"-key", keySchemaStr); err != nil {
			return fmt.Errorf("failed to get key schema id: %w", err)
		}
	}
	return nil
}

func (o *Kafka) getSchemaID(subject string, schemaStr string) ([]byte, error) {
	var (
		err    error
		schema *srclient.Schema
	)
	for _, host := range o.SchemaRegistries {
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

func (o *Kafka) Write(ctx context.Context, dm *types.DnstapMessage) error {
	var err error
	var v, k sarama.Encoder
	switch o.OutputType {
	case OutputTypePtoroBuf:
		k = sarama.ByteEncoder(o.Key)
		v = sarama.ByteEncoder(dm.GetRaw())
	case OutputTypeAvero:
		mapString, err := dm.ConvertV1Flat()
		if err != nil {
			return err
		}
		if v, err = o.GetEncoder(mapString, o.valueCodec, o.valueSchemaID); err != nil {
			return err
		}
		if k, err = o.GetEncoder(o.Key, o.keyCodec, o.keySchemaID); err != nil {
			return err
		}
	case OutputTypeJSON:
		buf, err := dm.ConvertV1JSONWithFilter(*o.OutputFilters)
		if err != nil {
			return err
		}
		k = sarama.StringEncoder(o.Key)
		v = sarama.StringEncoder(buf)
	}

	msg := &sarama.ProducerMessage{
		Topic: o.Topic,
		Key:   k,
		Value: v,
	}
	_, _, err = o.producer.SendMessage(msg)

	return err
}

func (o *Kafka) Close(context.Context) {
	o.producer.Close()
}

func (p *Kafka) MaxConcurrent() uint {
	return math.MaxUint32
}
