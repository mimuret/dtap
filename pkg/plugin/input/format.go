package input

import (
	"strings"

	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v3/pkg/types"
)

const DefaultFormat = "DNSTAP"

var (
	registry = map[string]NewFormatFunc{}
)

type NewFormatFunc func(p types.InputPlugin, options *framestream.DecoderOptions) *InputServer

func RegisterFormat(format string, newFunc NewFormatFunc) {
	registry[strings.ToLower(format)] = newFunc
}

func NewInputServer(p types.InputPlugin, format string, options *framestream.DecoderOptions) *InputServer {
	if format == "" {
		format = DefaultFormat
	}

	newFunc := registry[strings.ToLower(format)]
	if newFunc == nil {
		return nil
	}
	return newFunc(p, options)
}
