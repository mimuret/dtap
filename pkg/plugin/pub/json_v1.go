package pub

import (
	"bytes"
	"sync"

	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

const (
	FormatV1JSON Format = "JSON/V1"
)

var _ Publisher = &JsonV1Publisher{}

type JsonV1Publisher struct {
	sync.Mutex
	buf     *bytes.Buffer
	handler PublisherHandler
	maxSize int

	intervalFlusher *intervalFlusher

	writeSize  int
	writeState writeState
	writeCount int
}

func NewJsonV1Publisher(maxSize int, intervalSec uint, handler PublisherHandler) Publisher {
	buf := make([]byte, 0, maxSize)
	return &JsonV1Publisher{
		buf:             bytes.NewBuffer(buf),
		handler:         handler,
		maxSize:         maxSize,
		intervalFlusher: newIntervalFlusher(intervalSec),
	}
}

func (f *JsonV1Publisher) Start() {
	f.intervalFlusher.Start(f)
}

func (f *JsonV1Publisher) reset() {
	f.buf.Reset()
	f.writeState = writeStateInit
	f.writeSize = 0
	f.writeCount = 0
}

func (f *JsonV1Publisher) Write(dm *types.DnstapMessage) error {
	f.Lock()
	defer f.Unlock()
	if err := f.write(dm); err != nil {
		f.reset()
		return err
	}
	return nil
}

func (f *JsonV1Publisher) write(dm *types.DnstapMessage) error {
	data, err := dm.ConvertV1JSON()
	if err != nil {
		return errors.Wrap(err, "failed to convert json")
	}
	if f.writeSize+len(data)+2 > f.maxSize {
		err := f.Publish()
		if err != nil {
			return errors.Wrap(err, "failed to publish message")
		}
	}

	pre := byte(',')
	if f.writeState == writeStateInit {
		pre = '['
	}
	if err := f.buf.WriteByte(pre); err != nil {
		return errors.Wrap(err, "failed to write separator")
	}
	f.writeState = writeStateActive

	n, err := f.buf.Write(data)
	f.writeSize += n + 1
	f.writeCount++
	return errors.Wrap(err, "failed to write message")
}

func (f *JsonV1Publisher) Publish() error {
	if f.writeCount == 0 {
		return nil
	}
	if err := f.buf.WriteByte(']'); err != nil {
		return errors.Wrap(err, "failed to close message")
	}
	f.writeSize += 1
	data := f.buf.Bytes()
	err := f.handler.Publish(data[:f.writeSize])
	if err != nil {
		return errors.Wrap(err, "publish error")
	}
	f.reset()
	return nil
}

func (f *JsonV1Publisher) Close() error {
	f.intervalFlusher.Close()
	return f.Publish()
}

func init() {
	RegisterPublisher(FormatV1JSON, NewJsonV1Publisher)
}
