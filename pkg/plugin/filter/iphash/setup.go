package iphash

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const PLUGIN_NAME = "iphash"

func init() {
	_ = registry.RegisterFilterPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	fp := &IPHash{
		FilterBlock:            *cfg,
		QueryAddressEnabled:    true,
		ResponseAddressEnabled: true,
	}
	diags := gohcl.DecodeBody(cfg.Body, nil, fp)
	if diags.HasErrors() {
		return nil, plugin.PluginError(fp, "failed to setup err: %w", errors.Join(diags.Errs()...))
	}

	if fp.Salt == "" {
		return nil, plugin.PluginError(fp, "missing parameter Salts")
	}
	if !fp.QueryAddressEnabled && !fp.ResponseAddressEnabled {
		return nil, plugin.PluginError(fp, "QueryAddressEnabled and ResponseAddressEnabled are both false")
	}
	return fp, nil
}

var _ types.FilterPlugin = &IPHash{}

// The IPHash plugin creates QueryAddressHash and ResponseAddressHash labels from query and response addresses.

// Example HCL configuration:
// ```hcl
//
//	filter "iphash" "default" {
//		 salt = "my-secret-salt"
//	}
//
// ```
type IPHash struct {
	config.FilterBlock
	// salt for creating hash
	Salt string `hcl:"salt"`
	// If QueryAddressEnabled is true, create a QueryAddressHash label. Default is true.
	QueryAddressEnabled bool `hcl:"query_address_enabled,optional"`
	// If ResponseAddressEnabled is true, create a ResponseAddressHash label. Default is true.
	ResponseAddressEnabled bool `hcl:"response_address_enabled,optional"`
}

func (f *IPHash) Filter(ctx context.Context, t *types.DnstapMessage) *types.DnstapMessage {
	dt := t.GetDnstap()
	if dt.Message == nil {
		return t
	}
	t = t.DeepCopy()
	dt = t.GetDnstap()
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
