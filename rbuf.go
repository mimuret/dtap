package dtap

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type RBuf struct {
	channel     chan []byte
	mux         sync.Mutex
	inCounter   prometheus.Counter
	lostCounter prometheus.Counter
}

func NewRbuf(size uint, inCounter prometheus.Counter, lostCounter prometheus.Counter) *RBuf {
	rbuf := &RBuf{
		channel:     make(chan []byte, size),
		mux:         sync.Mutex{},
		inCounter:   inCounter,
		lostCounter: lostCounter,
	}
	return rbuf
}

func (r *RBuf) Read() <-chan []byte {
	return r.channel
}

func (r *RBuf) Write(b []byte) {
	r.mux.Lock()
	select {
	case r.channel <- b:
		r.inCounter.Inc()
	default:
		r.lostCounter.Inc()
		r.inCounter.Inc()
		<-r.channel
		r.channel <- b
	}
	r.mux.Unlock()
}

func (r *RBuf) Close() {
	close(r.channel)
}
