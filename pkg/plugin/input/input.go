package input

import (
	"io"
	"net"
	"sync"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/plugin/pub"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

const FormatDNSTAP Format = "DNSTAP"
const FormatDtapFrame Format = "DtapFrame"

var (
	TotalDecordError = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dtap_input_error_frame_total",
		Help: "The total number of input error frames",
	})
)

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
	ic                *types.InputContext
}

func NewDnstapInputServer(options *framestream.DecoderOptions, ic *types.InputContext) *InputServer {
	if options == nil {
		options = &framestream.DecoderOptions{
			Bidirectional: true,
		}
	}
	if options.ContentType == nil {
		options.ContentType = dnstap.FSContentType
	}
	return &InputServer{
		DecoderOptions:    options,
		ic:                ic,
		connectionManager: newConnectionManager(),
		unmarshaler:       types.NewDnstapMessage,
	}
}

func NewDtapFrameInputServer(options *framestream.DecoderOptions, ic *types.InputContext) *InputServer {
	if options == nil {
		options = &framestream.DecoderOptions{
			Bidirectional: true,
		}
	}
	if options.ContentType == nil {
		options.ContentType = pub.DtapFrameFSContentType
	}
	return &InputServer{
		DecoderOptions:    options,
		ic:                ic,
		connectionManager: newConnectionManager(),
		unmarshaler:       types.NewDnstapMessageFromDtapFrameRaw,
	}
}

func (i *InputServer) Serve(ln net.Listener, buf types.Writer) error {
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
			if err := i.Read(conn, buf); err != nil {
				TotalDecordError.Inc()
				i.ic.Logger.Debug("input error", zap.Error(err))
			}
			i.connectionManager.remove(conn)
			wg.Done()
		}(conn)
	}
}

func (i *InputServer) Read(r io.Reader, buf types.Writer) error {
	decoder, err := framestream.NewDecoder(r, i.DecoderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to create fstrm decoder")
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
			return errors.Wrap(err, "failed to decode DNSTAP message")
		}
		dm, err := i.unmarshaler(bs)
		if err != nil {
			return errors.Wrap(err, "failed to create dnstap message")
		}
		buf.Write(dm)
	}
	return nil
}

func init() {
	RegisterFormat(FormatDNSTAP, NewDnstapInputServer)
	RegisterFormat(FormatDtapFrame, NewDtapFrameInputServer)
}
