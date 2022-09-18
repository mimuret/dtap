package pub

import "strings"

type Format string

var (
	registry = map[Format]NewFormatFunc{}
)

type NewFormatFunc func(maxSize int, handler PublisherHandler) Publisher

func RegisterPublisher(f Format, newFunc NewFormatFunc) {
	f = Format(strings.ToUpper(string(f)))
	registry[f] = newFunc
}

func NewPublisher(f Format, maxSize int, handler PublisherHandler) Publisher {
	f = Format(strings.ToUpper(string(f)))
	newFunc := registry[f]
	if newFunc == nil {
		return nil
	}
	return newFunc(maxSize, handler)
}
