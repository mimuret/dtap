package input

import (
	"strings"

	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/types"
)

const DefaultFormat = "DNSTAP"

// The file format can be either "DNSTAP" or "DTAPFrame".
// "DNSTAP" is the normal DNSTAP format.
// "DTAPFrame" is a DNSTAP format with a label added.
// For details, https://github.com/mimuret/dtap/blob/v2/proto/dtap_frame.proto.
type FormatMeta struct {
	Format string
}

func (f FormatMeta) GetFormat() string {
	return strings.ToUpper(f.Format)
}

var (
	registry = map[string]NewFormatFunc{}
)

type PluginWithFormat interface {
	types.Plugin
	GetFormat() string
}

type NewFormatFunc func(p PluginWithFormat, options *framestream.DecoderOptions) *InputServer

func RegisterFormat(f FormatMeta, newFunc NewFormatFunc) {
	registry[f.GetFormat()] = newFunc
}

func NewInputServer(p PluginWithFormat, options *framestream.DecoderOptions) *InputServer {
	newFunc := registry[p.GetFormat()]
	if newFunc == nil {
		return nil
	}
	return newFunc(p, options)
}
