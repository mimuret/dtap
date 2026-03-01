package mask

import (
	"context"
	"errors"
	"net"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/mimuret/dtap/v3/pkg/config"
	"github.com/mimuret/dtap/v3/pkg/plugin"
	"github.com/mimuret/dtap/v3/pkg/plugin/registry"
	"github.com/mimuret/dtap/v3/pkg/types"
	"go.uber.org/zap"
)

const PLUGIN_NAME = "mask"

func init() {
	_ = registry.RegisterFilterPlugin(PLUGIN_NAME, Setup)
}

func Setup(cfg *config.FilterBlock) (types.FilterPlugin, error) {
	fp := &Mask{
		FilterBlock:            *cfg,
		MaskLen4:               22,
		MaskLen6:               40,
		QueryAddressEnabled:    true,
		ResponseAddressEnabled: true,
	}
	// Decode the HCL body into the Relabel struct.
	diags := gohcl.DecodeBody(cfg.Body, nil, fp)
	if diags.HasErrors() {
		// Return an error if there are issues decoding the HCL body.
		return nil, plugin.PluginError(fp, "failed to setup Relabel plugin: %w", errors.Join(diags.Errs()...))
	}
	if fp.MaskLen4 > 32 {
		return nil, errors.New("MaskLen4 must be between 0 and 32")
	}
	if fp.MaskLen6 > 128 {
		return nil, errors.New("MaskLen6 must be between 0 and 128")
	}
	fp.mask4 = net.CIDRMask(int(fp.MaskLen4), 32)
	fp.mask6 = net.CIDRMask(int(fp.MaskLen6), 128)

	return fp, nil
}

var _ types.FilterPlugin = &Mask{}

// Mask plug-ins mask IP addresses.
// Example HCL configuration:
//
//	filter "mask" "default" {
//	  mask_len4 = 22
//	  mask_len6 = 40
//	  query_address_enabled = true
//	  response_address_enabled = true
//	}
type Mask struct {
	config.FilterBlock
	// PrefixLength value to mask ipv4 address. default is 22.
	// MaskLen4: Default is 22 (applies to IPv4 addresses).
	MaskLen4 uint8 `hcl:"mask_len4,optional"`
	// PrefixLength value to mask ipv6 address. default is 40.
	// MaskLen6: Default is 40 (applies to IPv6 addresses).
	MaskLen6 uint8 `hcl:"mask_len6,optional"`
	// If QueryAddressEnabled is true, mask QueryAddress. defaut is true
	QueryAddressEnabled bool `hcl:"query_address_enabled,optional"`
	// If ResponseAddressEnabled is true, mask ResponseAddress. defaut is true
	ResponseAddressEnabled bool `hcl:"response_address_enabled,optional"`
	mask4                  net.IPMask
	mask6                  net.IPMask
}

func (f *Mask) maskAddress(address []byte, mask net.IPMask, isIPv6 bool) []byte {
	if address == nil {
		return nil
	}
	ip := net.IP(address).Mask(mask)
	if isIPv6 {
		return ip.To16()
	}
	return ip.To4()
}

func (f *Mask) Filter(ctx context.Context, t *types.DnstapMessage) *types.DnstapMessage {
	dt := t.GetDnstap()
	if dt.Message == nil {
		return t
	}
	t = t.DeepCopy()
	dt = t.GetDnstap()
	isIPv6 := *dt.Message.SocketFamily == dnstap.SocketFamily_INET6
	if f.QueryAddressEnabled {
		dt.Message.QueryAddress = f.maskAddress(dt.Message.QueryAddress, f.getMask(isIPv6), isIPv6)
	}
	if f.ResponseAddressEnabled {
		dt.Message.ResponseAddress = f.maskAddress(dt.Message.ResponseAddress, f.getMask(isIPv6), isIPv6)
	}
	if err := t.UpdateFromDnstap(dt); err != nil {
		ctxzap.Error(ctx, "failed to update DnstapMessage", zap.Error(err))
		return nil
	}
	return t
}

func (f *Mask) getMask(isIPv6 bool) net.IPMask {
	if isIPv6 {
		return f.mask6
	}
	return f.mask4
}
