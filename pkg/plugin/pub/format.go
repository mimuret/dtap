package pub

import "strings"

const DefaultFormat = "JSON/V1"

type Format string

var (
	registry = map[Format]NewFormatFunc{}
)

type NewFormatFunc func(maxSize int, intervalSec uint, handler PublisherHandler) Publisher

func RegisterPublisher(f Format, newFunc NewFormatFunc) {
	f = Format(strings.ToUpper(string(f)))
	registry[f] = newFunc
}

func NewPublisher(f Format, maxSize int, intervalSec uint, handler PublisherHandler) Publisher {
	f = Format(strings.ToUpper(string(f)))
	newFunc := registry[f]
	if newFunc == nil {
		return nil
	}
	return newFunc(maxSize, intervalSec, handler)
}
