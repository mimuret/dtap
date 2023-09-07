package pub

import "time"

type intervalFlusher struct {
	sec     uint
	stopCh  chan struct{}
	closeCh chan struct{}
}

func newIntervalFlusher(sec uint) *intervalFlusher {
	return &intervalFlusher{
		sec:     sec,
		stopCh:  make(chan struct{}),
		closeCh: make(chan struct{}),
	}
}

func (i *intervalFlusher) Start(p Publisher) {
	go func() {
		ticker := time.NewTicker(time.Duration(i.sec) * time.Second)
		defer ticker.Stop()
	LOOP:
		for {
			select {
			case <-ticker.C:
				_ = p.Publish()
			case <-i.stopCh:
				break LOOP
			}
		}
		close(i.closeCh)
	}()
}

func (i *intervalFlusher) Close() {
	close(i.stopCh)
	<-i.closeCh
}
