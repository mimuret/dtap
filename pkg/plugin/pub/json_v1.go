package pub

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mimuret/dtap/v3/pkg/types"
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

func NewJsonV1Publisher(maxSize int, interval time.Duration, handler PublisherHandler) Publisher {
	buf := make([]byte, 0, maxSize)
	return &JsonV1Publisher{
		buf:             bytes.NewBuffer(buf),
		handler:         handler,
		maxSize:         maxSize,
		intervalFlusher: newIntervalFlusher(interval),
	}
}

func (f *JsonV1Publisher) Start(ctx context.Context) {
	f.intervalFlusher.Start(ctx, f)
}

func (f *JsonV1Publisher) reset() {
	f.buf.Reset()
	f.writeState = writeStateInit
	f.writeSize = 0
	f.writeCount = 0
}

func (f *JsonV1Publisher) Write(ctx context.Context, dm *types.DnstapMessage) error {
	f.Lock()
	defer f.Unlock()
	if err := f.write(ctx, dm); err != nil {
		f.reset()
		return err
	}
	return nil
}

func (f *JsonV1Publisher) write(ctx context.Context, dm *types.DnstapMessage) error {
	data, err := dm.ConvertV1JSON()
	if err != nil {
		return fmt.Errorf("failed to convert json: %w", err)
	}
	if f.writeSize+len(data)+2 > f.maxSize {
		err := f.Publish(ctx)
		if err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}
	}

	pre := byte(',')
	if f.writeState == writeStateInit {
		pre = '['
	}
	if err := f.buf.WriteByte(pre); err != nil {
		return fmt.Errorf("failed to write separator: %w", err)
	}
	f.writeState = writeStateActive

	n, err := f.buf.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	f.writeSize += n + 1
	f.writeCount++
	return nil
}

func (f *JsonV1Publisher) Publish(ctx context.Context) error {
	if f.writeCount == 0 {
		return nil
	}
	if err := f.buf.WriteByte(']'); err != nil {
		return fmt.Errorf("failed to close message: %w", err)
	}
	f.writeSize += 1
	data := f.buf.Bytes()
	err := f.handler.Publish(ctx, data[:f.writeSize])
	if err != nil {
		return fmt.Errorf("publish error: %w", err)
	}
	f.reset()
	return nil
}

func (f *JsonV1Publisher) Close(ctx context.Context) error {
	f.intervalFlusher.Close(ctx)
	return f.Publish(ctx)
}

func init() {
	RegisterPublisher(FormatV1JSON, NewJsonV1Publisher)
}
