package pub

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v3/pkg/types"
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

	intervalFlusher *intervalFlusher
	marshaler       FstrmPublisherMarshaler
	writer          *framestream.Writer
	writeSize       int
	writeState      writeState
	writeCount      int
}

func NewFstrmPublisher(maxSize int, interval time.Duration, handler PublisherHandler) *FstrmPublisher {
	buf := make([]byte, 0, maxSize)
	return &FstrmPublisher{
		buf:             bytes.NewBuffer(buf),
		handler:         handler,
		maxSize:         maxSize,
		intervalFlusher: newIntervalFlusher(interval),
	}
}

func NewFstrmDNSTAPPublisher(maxSize int, interval time.Duration, handler PublisherHandler) Publisher {
	publisher := NewFstrmPublisher(maxSize, interval, handler)
	publisher.marshaler = func(dm *types.DnstapMessage) ([]byte, error) {
		return dm.GetRaw(), nil
	}
	publisher.contentType = dnstap.FSContentType
	return publisher
}

func NewFstrmDtapFramePublisher(maxSize int, interval time.Duration, handler PublisherHandler) Publisher {
	publisher := NewFstrmPublisher(maxSize, interval, handler)
	publisher.marshaler = func(dm *types.DnstapMessage) ([]byte, error) {
		return proto.Marshal(dm.ToDtapFrame())
	}
	publisher.contentType = DtapFrameFSContentType
	return publisher
}

func (f *FstrmPublisher) Start(ctx context.Context) {
	f.intervalFlusher.Start(ctx, f)
}

func (f *FstrmPublisher) reset() {
	f.buf.Reset()
	f.writeState = writeStateInit
	f.writeSize = 0
	f.writeCount = 0
}

func (f *FstrmPublisher) Write(ctx context.Context, dm *types.DnstapMessage) error {
	f.Lock()
	defer f.Unlock()
	if err := f.write(ctx, dm); err != nil {
		f.reset()
		return err
	}
	return nil
}

func (f *FstrmPublisher) write(ctx context.Context, dm *types.DnstapMessage) error {
	var err error
	data, err := f.marshaler(dm)
	if err != nil {
		// broken message is not processed, return nil error
		return nil
	}
	if f.writeSize+4+len(data)+DnstapFstrmControlHeaderSize > f.maxSize {
		if err := f.publish(ctx); err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}
	}
	if f.writeState == writeStateInit {
		f.writer, err = framestream.NewWriter(f.buf, &framestream.WriterOptions{
			ContentTypes:  [][]byte{f.contentType},
			Bidirectional: false,
		})
		if err != nil {
			return fmt.Errorf("failed to create writer: %w", err)
		}
		f.writeState = writeStateActive
		f.writeSize = DnstapFstrmControlHeaderSize
		f.writeCount = 0
	}
	n, err := f.writer.WriteFrame(data)
	if err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}
	f.writeSize += n + DnstapFstrmMsgHeaderSize
	f.writeCount++
	return nil
}

func (f *FstrmPublisher) Publish(ctx context.Context) error {
	f.Lock()
	defer f.Unlock()
	return f.publish(ctx)
}

func (f *FstrmPublisher) publish(ctx context.Context) error {
	if f.writeState != writeStateActive {
		return nil
	}
	if err := f.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush fstrm: %w", err)
	}
	if err := f.writer.Close(); err != nil {
		return fmt.Errorf("failed to close fstrm: %w", err)
	}
	f.writeSize += DnstapFstrmControlHeaderSize
	if f.writeCount == 0 {
		return nil
	}
	data := f.buf.Bytes()
	err := f.handler.Publish(ctx, data[:f.writeSize])
	if err != nil {
		return fmt.Errorf("publish error: %w", err)
	}
	f.reset()
	return nil
}

func (f *FstrmPublisher) Close(ctx context.Context) error {
	f.intervalFlusher.Close(ctx)
	return f.Publish(ctx)
}

func init() {
	RegisterPublisher(FormatDNSTAP, NewFstrmDNSTAPPublisher)
	RegisterPublisher(FormatDtapFrame, NewFstrmDtapFramePublisher)
}
