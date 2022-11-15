package pub

import (
	"bytes"
	"sync"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
)

const FormatDNSTAP Format = "DNSTAP"

const (
	DnstapFstrmControlHeaderSize = 42
	DnstapFstrmMsgHeaderSize     = 4
)

var _ Publisher = &DnstapPublisher{}

type DnstapPublisher struct {
	sync.Mutex
	buf     *bytes.Buffer
	handler PublisherHandler
	maxSize int

	interval   *intervalSec
	writer     *framestream.Writer
	writeSize  int
	writeState writeState
	writeCount int
}

func NewDnstapPublisher(maxSize int, intervalSec uint, handler PublisherHandler) Publisher {
	buf := make([]byte, 0, maxSize)
	return &DnstapPublisher{
		buf:      bytes.NewBuffer(buf),
		handler:  handler,
		maxSize:  maxSize,
		interval: newIntervalSec(intervalSec),
	}
}

func (f *DnstapPublisher) Start() {
	f.interval.Start(f)
}

func (f *DnstapPublisher) reset() {
	f.buf.Reset()
	f.writeState = writeStateInit
	f.writeSize = 0
	f.writeCount = 0
}

func (f *DnstapPublisher) Write(dm *types.DnstapMessage) error {
	f.Lock()
	defer f.Unlock()
	if err := f.write(dm); err != nil {
		f.reset()
		return err
	}
	return nil
}

func (f *DnstapPublisher) write(dm *types.DnstapMessage) error {
	var err error
	data := dm.GetRaw()
	if f.writeSize+4+len(data)+DnstapFstrmControlHeaderSize > f.maxSize {
		if err := f.Publish(); err != nil {
			return errors.Wrap(err, "failed to publish message")
		}
	}
	if f.writeState == writeStateInit {
		f.writer, err = framestream.NewWriter(f.buf, &framestream.WriterOptions{
			ContentTypes:  [][]byte{dnstap.FSContentType},
			Bidirectional: false,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create writer")
		}
		f.writeState = writeStateActive
		f.writeSize = DnstapFstrmControlHeaderSize
		f.writeCount = 0
	}
	n, err := f.writer.WriteFrame(data)
	f.writeSize += n + DnstapFstrmMsgHeaderSize
	f.writeCount++
	return errors.Wrap(err, "failed to write message")
}

func (f *DnstapPublisher) Publish() error {
	if err := f.writer.Flush(); err != nil {
		return errors.Wrap(err, "failed to flush fstrm")
	}
	if err := f.writer.Close(); err != nil {
		return errors.Wrap(err, "failed to close fstrm")
	}
	f.writeSize += DnstapFstrmControlHeaderSize
	if f.writeCount == 0 {
		return nil
	}
	data := f.buf.Bytes()
	err := f.handler.Publish(data[:f.writeSize])
	if err != nil {
		return errors.Wrap(err, "publish error")
	}
	f.reset()
	return nil
}

func (f *DnstapPublisher) Close() error {
	f.interval.Close()
	return f.Publish()
}

func init() {
	RegisterPublisher(FormatDNSTAP, NewDnstapPublisher)
}
