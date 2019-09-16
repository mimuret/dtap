/*
 * Copyright (c) 2018 Manabu Sonoda
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

package dtap_test

import (
	"bytes"
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mimuret/dtap"
	"github.com/stretchr/testify/assert"
)

func TestOutputNatsConfig(t *testing.T) {
	cfg := `[[InputUnix]]
Path="/var/log/unbound/dnstap.sock"
User="unbound"

[[OutputNats]]
	Host = "nats://host1:4242, nats://host2:4242,  nats://host3:4242"
	User = "cns"
	Password = "hogehoge"
	Subject = "query"
	[OutputNats.flat]
		IPv4Mask = 22
		IPv6Mask = 40
		EnableECS = true
		EnableHashIP = true
		IPHashSalt = "bb"
`
	b := bytes.NewBufferString(cfg)
	c, err := dtap.NewConfigFromReader(b)
	assert.NoError(t, err)
	assert.Equal(t, c.InputUnix[0].GetPath(), "/var/log/unbound/dnstap.sock")
	assert.Equal(t, c.InputUnix[0].GetUser(), "unbound")
	assert.Nil(t, c.InputUnix[0].Validate())

	assert.Equal(t, c.OutputNats[0].GetHost(), "nats://host1:4242, nats://host2:4242,  nats://host3:4242")
	assert.Equal(t, c.OutputNats[0].GetUser(), "cns")
	assert.Equal(t, c.OutputNats[0].GetPassword(), "hogehoge")
	assert.Equal(t, c.OutputNats[0].GetSubject(), "query")
	assert.Equal(t, c.OutputNats[0].Flat.GetEnableEcs(), true)
	assert.Equal(t, c.OutputNats[0].Flat.GetIPv4Mask(), net.CIDRMask(22, 32))
	assert.Equal(t, c.OutputNats[0].Flat.GetIPv6Mask(), net.CIDRMask(40, 128))

	assert.Equal(t, c.OutputNats[0].Flat.GetEnableHashIP(), true)

}

func TestFlatConfig(t *testing.T) {
	cfg := `[[InputUnix]]
Path="/var/log/unbound/dnstap.sock"
User="unbound"

[[OutputNats]]
	Host = "nats://host1:4242, nats://host2:4242,  nats://host3:4242"
	User = "cns"
	Password = "hogehoge"
	Subject = "query"
	[OutputNats.flat]
		IPv4Mask = 22
		IPv6Mask = 40
		EnableECS = true
		EnableHashIP = true
		IPHashSaltPath = "/tmp/salt.test"
`
	f, _ := os.Create("/tmp/salt.test")
	f.Write([]byte{10, 20, 30, 40})
	f.Close()
	b := bytes.NewBufferString(cfg)
	c, err := dtap.NewConfigFromReader(b)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.TODO())
	go c.OutputNats[0].Flat.WatchSalt(ctx)

	assert.Equal(t, c.OutputNats[0].Flat.GetEnableEcs(), true)
	assert.Equal(t, c.OutputNats[0].Flat.GetIPv4Mask(), net.CIDRMask(22, 32))
	assert.Equal(t, c.OutputNats[0].Flat.GetIPv6Mask(), net.CIDRMask(40, 128))
	assert.Equal(t, c.OutputNats[0].Flat.GetEnableHashIP(), true)
	assert.Equal(t, c.OutputNats[0].Flat.GetIPHashSalt(), []byte{10, 20, 30, 40})
	f, _ = os.Create("/tmp/salt.test")
	f.Write([]byte{20, 30, 40, 50})
	f.Close()
	time.Sleep(10 * time.Second)
	assert.Equal(t, c.OutputNats[0].Flat.GetIPHashSalt(), []byte{20, 30, 40, 50})
	cancel()
}
