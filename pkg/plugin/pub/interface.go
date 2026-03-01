package pub

import (
	"context"

	"github.com/mimuret/dtap/v3/pkg/types"
)

type PublisherHandler interface {
	Publish(context.Context, []byte) error
}

type ConsumerHandler interface {
	Consumer([][]byte) error
}

type Publisher interface {
	Write(context.Context, *types.DnstapMessage) error
	Start(context.Context)
	Close(context.Context) error
	Publish(context.Context) error
}
