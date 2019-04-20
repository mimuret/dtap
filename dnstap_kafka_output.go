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
	"net"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type DnstapKafkaOutput struct {
	config      *OutputKafkaConfig
	enc         *framestream.Encoder
	kafkaConfig *sarama.Config
	producer    sarama.AsyncProducer
	ipv4Mask    net.IPMask
	ipv6Mask    net.IPMask
}

func NewDnstapKafkaOutput(config *OutputKafkaConfig) *DnstapOutput {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Flush.Messages = 100
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Retry.Max = int(config.GetRetry())

	o := &DnstapKafkaOutput{
		config:      config,
		kafkaConfig: kafkaConfig,
		ipv4Mask:    net.CIDRMask(config.GetIPv4Mask(), 32),
		ipv6Mask:    net.CIDRMask(config.GetIPv6Mask(), 128),
	}
	return NewDnstapOutput(config.GetBufferSize(), o)
}

func (o *DnstapKafkaOutput) open() error {
	var err error
	o.producer, err = sarama.NewAsyncProducer(o.config.Hosts, o.kafkaConfig)
	if err != nil {
		return errors.Wrapf(err, "can't create kafka producer")
	}
	return nil
}

func (o *DnstapKafkaOutput) write(frame []byte) error {
	dt := &Dnstap{}
	if err := proto.Unmarshal(frame, dt); err != nil {
		return err
	}
	data, err := dt.Flat(o.ipv4Mask, o.ipv6Mask)
	if err != nil {
		return err
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return err
	}
	timestamp := time.Now().UnixNano()

	o.producer.Input() <- &sarama.ProducerMessage{
		Topic: o.config.GetTopic(),
		Key:   sarama.StringEncoder(strconv.FormatInt(timestamp, 10)),
		Value: sarama.StringEncoder(string(jsonStr)),
	}
	return nil
}

func (o *DnstapKafkaOutput) close() {
	o.producer.Close()
}
