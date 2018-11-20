package dtap

import (
	"sync"
)

type RBuf struct {
	values [][]byte
	size   uint
	rp     uint
	wp     uint
	rCh    chan []byte
	rmux   sync.Mutex
	wmux   sync.Mutex
}

func NewRbuf(size uint) *RBuf {
	rbuf := &RBuf{
		values: make([][]byte, size),
		size:   size,
		rp:     0,
		wp:     0,
		rCh:    make(chan []byte),
		wmux:   sync.Mutex{},
		rmux:   sync.Mutex{},
	}
	return rbuf
}

func (r *RBuf) Read() []byte {
	r.rmux.Lock()
	defer r.rmux.Lock()

	if r.rp != r.wp {
		r.rp++
		return r.values[r.rp]
	}

	return nil
}

func (r *RBuf) Write(b []byte) {
	r.wmux.Lock()
	defer r.wmux.Lock()

	r.values[r.wp] = b
	r.wp = (r.wp + 1) % r.size
}

func (r *RBuf) Close() {
	close(r.rCh)
}
