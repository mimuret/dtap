package mask

import (
	"errors"
	"net"

	"github.com/goccy/go-json"

	dnstap "github.com/dnstap/golang-dnstap"
	"github.com/mimuret/dtap/v2/pkg/plugin"
	"github.com/mimuret/dtap/v2/pkg/plugin/registry"
	"github.com/mimuret/dtap/v2/pkg/types"
)

func init() {
	_ = registry.RegisterFilterPlugin("mask", Setup)
}

func Setup(raw json.RawMessage) (types.FilterPlugin, error) {
	fp := &Mask{
		MaskLen4:               22,
		MaskLen6:               40,
		QueryAddressEnabled:    true,
		ResponseAddressEnabled: true,
	}
	if err := json.Unmarshal(raw, fp); err != nil {
		return nil, err
	}
	if fp.MaskLen4 > 32 {
		return nil, errors.New("invalid value MaskLen4")
	}
	if fp.MaskLen6 > 128 {
		return nil, errors.New("invalid value MaskLen6")
	}
	fp.mask4 = net.CIDRMask(int(fp.MaskLen4), 32)
	fp.mask6 = net.CIDRMask(int(fp.MaskLen6), 128)

	return fp, nil
}

var _ types.FilterPlugin = &Mask{}

// Mask plug-ins mask IP addresses.
type Mask struct {
	plugin.PluginCommon
	// PrefixLength value to mask ipv4 address. default is 22.
	MaskLen4 uint8
	// PrefixLength value to mask ipv6 address. default is 40.
	MaskLen6 uint8
	// If QueryAddressEnabled is true, mask QueryAddress. defaut is true
	QueryAddressEnabled bool
	// If ResponseAddressEnabled is true, mask ResponseAddress. defaut is true
	ResponseAddressEnabled bool
	mask4                  net.IPMask
	mask6                  net.IPMask
}

func (f *Mask) Filter(t *types.DnstapMessage) *types.DnstapMessage {
	dt := t.GetDnstap()
	if dt.Message == nil {
		return t
	}
	if *dt.Message.SocketFamily == dnstap.SocketFamily_INET {
		if f.QueryAddressEnabled && dt.Message.QueryAddress != nil {
			dt.Message.QueryAddress = []byte(net.IP(dt.Message.QueryAddress).Mask(f.mask4).To4())
		}
		if f.ResponseAddressEnabled && dt.Message.ResponseAddress != nil {
			dt.Message.ResponseAddress = []byte(net.IP(dt.Message.ResponseAddress).Mask(f.mask4).To4())
		}
	} else if *dt.Message.SocketFamily == dnstap.SocketFamily_INET6 {
		if f.QueryAddressEnabled && dt.Message.QueryAddress != nil {
			dt.Message.QueryAddress = []byte(net.IP(dt.Message.QueryAddress).Mask(f.mask6).To16())
		}
		if f.ResponseAddressEnabled && dt.Message.ResponseAddress != nil {
			dt.Message.ResponseAddress = []byte(net.IP(dt.Message.ResponseAddress).Mask(f.mask6).To16())
		}
	}
	if err := t.UpdateFromDnstap(dt); err != nil {
		return nil
	}
	return t
}
