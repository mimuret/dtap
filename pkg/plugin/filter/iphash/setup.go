package iphash

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"

	"github.com/goccy/go-json"

	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
)

func init() {
	_ = registry.RegisterFilterPlugin("iphash", Setup)
}

func Setup(raw json.RawMessage) (types.FilterPlugin, error) {
	fp := &IPHash{
		QueryAddressEnabled:    true,
		ResponseAddressEnabled: true,
	}
	if err := json.Unmarshal(raw, fp); err != nil {
		return nil, err
	}
	if fp.Salt == "" {
		return nil, errors.New("missing parameter Salt")
	}
	return fp, nil
}

var _ types.FilterPlugin = &IPHash{}

type IPHash struct {
	plugin.PluginCommon
	Salt                   string
	QueryAddressEnabled    bool
	ResponseAddressEnabled bool
}

func (f *IPHash) Filter(t *types.DnstapMessage) *types.DnstapMessage {
	dt := t.GetDnstap()
	if dt.Message == nil {
		return t
	}
	if f.QueryAddressEnabled && dt.Message.QueryAddress != nil {
		bs := make([]byte, 0, len([]byte(f.Salt))+16)
		copy(bs, []byte(f.Salt))
		bs = append(bs, net.IP(dt.Message.QueryAddress).To16()...)
		h := sha256.Sum256(bs)
		t.Labels["QueryAddressHash"] = hex.EncodeToString(h[:])
	}
	if f.ResponseAddressEnabled && dt.Message.ResponseAddress != nil {
		bs := make([]byte, 0, len([]byte(f.Salt))+16)
		copy(bs, []byte(f.Salt))
		bs = append(bs, net.IP(dt.Message.ResponseAddress).To16()...)
		h := sha256.Sum256(bs)
		t.Labels["ResponseAddressHash"] = hex.EncodeToString(h[:])
	}
	return t
}
