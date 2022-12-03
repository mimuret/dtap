package input

import (
	"strings"

	framestream "github.com/farsightsec/golang-framestream"
)

const DefaultFormat = "DNSTAP"

type Format string

var (
	registry = map[Format]NewFormatFunc{}
)

type NewFormatFunc func(options *framestream.DecoderOptions) *InputServer

func RegisterFormat(f Format, newFunc NewFormatFunc) {
	f = Format(strings.ToUpper(string(f)))
	registry[f] = newFunc
}

func NewInputServer(f Format, options *framestream.DecoderOptions) *InputServer {
	f = Format(strings.ToUpper(string(f)))
	newFunc := registry[f]
	if newFunc == nil {
		return nil
	}
	return newFunc(options)
}
