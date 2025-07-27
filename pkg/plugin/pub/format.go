package pub

import (
	"strings"
	"time"
)

const DefaultFormat = "JSON/V1"

type Format string

var (
	registry = map[Format]NewFormatFunc{}
)

type NewFormatFunc func(maxSize int, interval time.Duration, handler PublisherHandler) Publisher

func RegisterPublisher(f Format, newFunc NewFormatFunc) {
	f = Format(strings.ToUpper(string(f)))
	registry[f] = newFunc
}

func NewPublisher(f Format, maxSize int, interval time.Duration, handler PublisherHandler) Publisher {
	f = Format(strings.ToUpper(string(f)))
	newFunc := registry[f]
	if newFunc == nil {
		return nil
	}
	return newFunc(maxSize, interval, handler)
}
