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
	"encoding/json"
	"fmt"
	"io/ioutil"

	kafka "github.com/dangkaka/go-kafka-avro"
	"github.com/rakyll/statik/fs"

	"github.com/Shopify/sarama"
	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/golang/protobuf/proto"
	_ "github.com/mimuret/dtap/statik"
	"github.com/pkg/errors"
)

var schemaStr string

func init() {
	var err error
	statikFS, _ := fs.New()
	f, _ := statikFS.Open("/flat.avsc")
	b, _ := ioutil.ReadAll(f)
	schemaStr = string(b)
}

type KafkaClient interface {
	Add(string, string, []byte, []byte) error
}

type DnstapKafkaOutput struct {
	config      *OutputKafkaConfig
	kafkaConfig *sarama.Config
	producer    KafkaClient
}

func NewDnstapKafkaOutput(config *OutputKafkaConfig, params *DnstapOutputParams) *DnstapOutput {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Retry.Max = int(config.GetRetry())

	params.Handler = &DnstapKafkaOutput{
		config:      config,
		kafkaConfig: kafkaConfig,
	}
	return NewDnstapOutput(params)
}

func (o *DnstapKafkaOutput) open() error {
	var err error
	o.producer, err = kafka.NewAvroProducer(o.config.Hosts, o.config.SchemaRegistries)
	if err != nil {
		return errors.Wrapf(err, "can't create kafka producer")
	}
	return nil
}

func (o *DnstapKafkaOutput) write(frame []byte) error {
	var k, v sarama.Encoder
	var bs []byte
	switch o.config.GetOutputType() {
	case "avro":
		dt := dnstap.Dnstap{}
		if err := proto.Unmarshal(frame, &dt); err != nil {
			return err
		}
		if o.config.GetOutputType() == "json" {
			bs, err := json.Marshal(dt)
			if err != nil {
				return err
			}
			k = sarama.StringEncoder(o.config.GetKey())
			v = sarama.StringEncoder(string(bs))
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
				bs, err := json.Marshal(data)
				if err != nil {
					return err
				}
				o.producer.Add(o.config.GetTopic(), schemaStr, []byte(o.config.GetKey()), bs)
				/*				mapString := data.ToMapString()
								log.Debug(mapString)
								binary, err := codec.BinaryFromNative(nil, mapString)
								if err != nil {
									return err
								}
								k = sarama.ByteEncoder(o.config.GetKey())
								v = sarama.ByteEncoder(binary)
				*/
			} else {
				bs, err := json.Marshal(data)
				if err != nil {
					return err
				}
				k = sarama.StringEncoder(o.config.GetKey())
				v = sarama.StringEncoder(string(bs))

			}
		}
	default:
		panic(fmt.Errorf("OutputType error: %s", o.config.GetOutputType()))
	}

	return nil
}

func (o *DnstapKafkaOutput) close() {
}
