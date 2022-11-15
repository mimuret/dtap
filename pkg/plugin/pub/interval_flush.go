package pub

import "time"

type intervalSec struct {
	sec     uint
	stopCh  chan struct{}
	closeCh chan struct{}
}

func newIntervalSec(sec uint) *intervalSec {
	return &intervalSec{
		sec:     sec,
		stopCh:  make(chan struct{}),
		closeCh: make(chan struct{}),
	}
}

func (i *intervalSec) Start(p Publisher) {
	go func() {
		ticker := time.NewTicker(time.Duration(i.sec) * time.Second)
	LOOP:
		for {
			select {
			case <-ticker.C:
				_ = p.Publish()
			case <-i.stopCh:
				break LOOP
			}
		}
	}()
	close(i.closeCh)
}

func (i *intervalSec) Close() {
	close(i.stopCh)
	<-i.closeCh
}
