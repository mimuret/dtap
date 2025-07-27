package pub

import (
	"context"
	"time"
)

type intervalFlusher struct {
	interval time.Duration
	stopCh   chan struct{}
	closeCh  chan struct{}
}

func newIntervalFlusher(interval time.Duration) *intervalFlusher {
	return &intervalFlusher{
		interval: interval,
		stopCh:   make(chan struct{}),
		closeCh:  make(chan struct{}),
	}
}

func (i *intervalFlusher) Start(ctx context.Context, p Publisher) {
	go func() {
		ticker := time.NewTicker(i.interval)
		defer ticker.Stop()
	LOOP:
		for {
			select {
			case <-ticker.C:
				_ = p.Publish(ctx)
			case <-i.stopCh:
				break LOOP
			}
		}
		close(i.closeCh)
	}()
}

func (i *intervalFlusher) Close(ctx context.Context) {
	close(i.stopCh)
	<-i.closeCh
}
