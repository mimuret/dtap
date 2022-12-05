package pub

import (
	"bytes"
	"sync"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const FormatDNSTAP Format = "DNSTAP"
const FormatDtapFrame Format = "DTAPFrame"

// FSContentType is the FrameStream content type for dnstap protobuf data.
var DtapFrameFSContentType = []byte("protobuf:dtap.DtapFrame")

const (
	DnstapFstrmControlHeaderSize = 42
	DnstapFstrmMsgHeaderSize     = 4
)

var _ Publisher = &FstrmPublisher{}

type FstrmPublisherMarshaler func(*types.DnstapMessage) ([]byte, error)

type FstrmPublisher struct {
	sync.Mutex
	buf         *bytes.Buffer
	handler     PublisherHandler
	maxSize     int
	contentType []byte

	interval   *intervalSec
	marshaler  FstrmPublisherMarshaler
	writer     *framestream.Writer
	writeSize  int
	writeState writeState
	writeCount int
}

func NewFstrmPublisher(maxSize int, intervalSec uint, handler PublisherHandler) *FstrmPublisher {
	buf := make([]byte, 0, maxSize)
	return &FstrmPublisher{
		buf:      bytes.NewBuffer(buf),
		handler:  handler,
		maxSize:  maxSize,
		interval: newIntervalSec(intervalSec),
	}
}

func NewFstrmDNSTAPPublisher(maxSize int, intervalSec uint, handler PublisherHandler) Publisher {
	publisher := NewFstrmPublisher(maxSize, intervalSec, handler)
	publisher.marshaler = func(dm *types.DnstapMessage) ([]byte, error) {
		return dm.GetRaw(), nil
	}
	publisher.contentType = dnstap.FSContentType
	return publisher
}

func NewFstrmDtapFramePublisher(maxSize int, intervalSec uint, handler PublisherHandler) Publisher {
	publisher := NewFstrmPublisher(maxSize, intervalSec, handler)
	publisher.marshaler = func(dm *types.DnstapMessage) ([]byte, error) {
		return proto.Marshal(dm.ToDtapFrame())
	}
	publisher.contentType = DtapFrameFSContentType
	return publisher
}

func (f *FstrmPublisher) Start() {
	f.interval.Start(f)
}

func (f *FstrmPublisher) reset() {
	f.buf.Reset()
	f.writeState = writeStateInit
	f.writeSize = 0
	f.writeCount = 0
}

func (f *FstrmPublisher) Write(dm *types.DnstapMessage) error {
	f.Lock()
	defer f.Unlock()
	if err := f.write(dm); err != nil {
		f.reset()
		return err
	}
	return nil
}

func (f *FstrmPublisher) write(dm *types.DnstapMessage) error {
	var err error
	data, err := f.marshaler(dm)
	if err != nil {
		return nil
	}
	if f.writeSize+4+len(data)+DnstapFstrmControlHeaderSize > f.maxSize {
		if err := f.publish(); err != nil {
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

func (f *FstrmPublisher) Publish() error {
	f.Lock()
	defer f.Unlock()
	return f.publish()
}

func (f *FstrmPublisher) publish() error {
	if f.writeState != writeStateActive {
		return nil
	}
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

func (f *FstrmPublisher) Close() error {
	f.interval.Close()
	<-f.interval.closeCh
	return f.Publish()
}

func init() {
	RegisterPublisher(FormatDNSTAP, NewFstrmDNSTAPPublisher)
	RegisterPublisher(FormatDtapFrame, NewFstrmDtapFramePublisher)
}
