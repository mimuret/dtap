package input

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/mimuret/dtap/v3/pkg/plugin/pub"
	"github.com/mimuret/dtap/v3/pkg/promauto"
	"github.com/mimuret/dtap/v3/pkg/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const FormatDNSTAP = "DNSTAP"
const FormatDtapFrame = "DtapFrame"

type connectionManager struct {
	sync.Mutex
	connections map[net.Conn]struct{}
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		connections: make(map[net.Conn]struct{}),
	}
}
func (c *connectionManager) register(conn net.Conn) {
	c.Lock()
	defer c.Unlock()
	c.connections[conn] = struct{}{}
}
func (c *connectionManager) remove(conn net.Conn) {
	c.Lock()
	defer c.Unlock()
	conn.Close()
	delete(c.connections, conn)
}
func (c *connectionManager) close() {
	for conn := range c.connections {
		conn.Close()
	}
}

type FstrmUnmarshaler func([]byte) (*types.DnstapMessage, error)

type InputServer struct {
	DecoderOptions    *framestream.DecoderOptions
	connectionManager *connectionManager
	unmarshaler       FstrmUnmarshaler

	fstrmDecordErrCount prometheus.Counter

	totalDecordErrorCount   prometheus.Counter
	msgDecordErrCount       prometheus.Counter
	unmarshalDecordErrCount prometheus.Counter
}

func newInputServer(p types.Plugin, options *framestream.DecoderOptions, unmarshaler FstrmUnmarshaler) *InputServer {
	is := &InputServer{
		connectionManager: newConnectionManager(),
		DecoderOptions:    options,
		unmarshaler:       unmarshaler,
	}
	is.fstrmDecordErrCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "fstrm_decord_errors_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	is.totalDecordErrorCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "read_frame_errors_total",
		Help:        "The total number of input error frames",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	is.msgDecordErrCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "message_decord_errors_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	is.unmarshalDecordErrCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "unmarshal_errors_total",
		ConstLabels: prometheus.Labels{"plugin": p.GetFullName()},
	})
	return is
}

func NewDnstapInputServer(p types.InputPlugin, options *framestream.DecoderOptions) *InputServer {
	if options == nil {
		options = &framestream.DecoderOptions{
			Bidirectional: true,
		}
	}
	if options.ContentType == nil {
		options.ContentType = dnstap.FSContentType
	}
	return newInputServer(p, options, types.NewDnstapMessage)
}

func NewDtapFrameInputServer(p types.InputPlugin, options *framestream.DecoderOptions) *InputServer {
	if options == nil {
		options = &framestream.DecoderOptions{
			Bidirectional: true,
		}
	}
	if options.ContentType == nil {
		options.ContentType = pub.DtapFrameFSContentType
	}
	return newInputServer(p, options, types.NewDnstapMessageFromDtapFrameRaw)
}

func (i *InputServer) Serve(ctx context.Context, forwader types.Forwarder, ln net.Listener) error {
	wg := sync.WaitGroup{}
	defer func() {
		i.connectionManager.close()
		wg.Wait()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		i.connectionManager.register(conn)
		wg.Add(1)
		go func(conn net.Conn) {
			if err := i.Read(ctx, forwader, conn); err != nil {
				i.totalDecordErrorCount.Inc()
				ctxzap.Debug(ctx, "input error", zap.Error(err))
			}

			i.connectionManager.remove(conn)
			wg.Done()
		}(conn)
	}
}

func (i *InputServer) Read(ctx context.Context, forwader types.Forwarder, r io.Reader) error {
	decoder, err := framestream.NewDecoder(r, i.DecoderOptions)
	if err != nil {
		i.fstrmDecordErrCount.Inc()
		return fmt.Errorf("failed to create fstrm decoder: %w", err)
	}
LOOP:
	for {
		bs, err := decoder.Decode()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break LOOP
			}
			if errors.Is(err, io.EOF) {
				break LOOP
			}
			i.fstrmDecordErrCount.Inc()
			return fmt.Errorf("failed to decode DNSTAP message: %w", err)
		}
		dm, err := i.unmarshaler(bytes.Clone(bs))
		if err != nil {
			i.unmarshalDecordErrCount.Inc()
			ctxzap.Debug(ctx, "input error", zap.Error(err), zap.Binary("bytes", bs))
			continue
		}
		forwader.Forward(dm)
	}
	return nil
}

func init() {
	RegisterFormat(FormatDNSTAP, NewDnstapInputServer)
	RegisterFormat(FormatDtapFrame, NewDtapFrameInputServer)
}
