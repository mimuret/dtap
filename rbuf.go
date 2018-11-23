package dtap

import (
	"sync"
)

type RBuf struct {
	channel chan []byte
	mux     sync.Mutex
}

func NewRbuf(size uint) *RBuf {
	rbuf := &RBuf{
		channel: make(chan []byte, size),
		mux:     sync.Mutex{},
	}
	return rbuf
}

func (r *RBuf) Read() <-chan []byte {
	return r.channel
}

func (r *RBuf) Write(b []byte) {
	select {
	case r.channel <- b:
	default:
		r.mux.Lock()
		<-r.channel
		r.channel <- b
		r.mux.Unlock()
	}
}

func (r *RBuf) Close() {
	close(r.channel)
}
