package pub

import "github.com/mimuret/dtap/v2/pkg/types"

type PublisherHandler interface {
	Publish([]byte) error
}

type ConsumerHandler interface {
	Consumer([][]byte) error
}

type Publisher interface {
	Write(*types.DnstapMessage) error
	Close() error
	Publish() error
}
