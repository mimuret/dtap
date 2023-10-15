package input

import (
	"io"
	"net"
	"sync"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/plugin/pub"
	"github.com/mimuret/dtap/v2/pkg/promauto"
	"github.com/mimuret/dtap/v2/pkg/types"
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

func newInputServer(p PluginWithFormat, options *framestream.DecoderOptions, unmarshaler FstrmUnmarshaler) *InputServer {
	is := &InputServer{
		connectionManager: newConnectionManager(),
		DecoderOptions:    options,
		unmarshaler:       unmarshaler,
	}
	is.fstrmDecordErrCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "fstrm_decord_errors_total",
		ConstLabels: prometheus.Labels{"ID": p.GetID()},
	})
	is.totalDecordErrorCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "read_frame_errors_total",
		Help:        "The total number of input error frames",
		ConstLabels: prometheus.Labels{"ID": p.GetID()},
	})
	is.msgDecordErrCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "message_decord_errors_total",
		ConstLabels: prometheus.Labels{"ID": p.GetID()},
	})
	is.unmarshalDecordErrCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "dtap",
		Subsystem:   "input_server",
		Name:        "unmarshal_errors_total",
		ConstLabels: prometheus.Labels{"ID": p.GetID()},
	})
	return is
}

func NewDnstapInputServer(p PluginWithFormat, options *framestream.DecoderOptions) *InputServer {
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

func NewDtapFrameInputServer(p PluginWithFormat, options *framestream.DecoderOptions) *InputServer {
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

func (i *InputServer) Serve(p PluginWithFormat, ln net.Listener, buf types.Writer, ic *types.InputContext) error {
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
			if err := i.Read(conn, buf, ic); err != nil {
				i.totalDecordErrorCount.Inc()
				ic.Logger.Debug("input error", zap.Error(err))
			}
			i.connectionManager.remove(conn)
			wg.Done()
		}(conn)
	}
}

func (i *InputServer) Read(r io.Reader, buf types.Writer, ic *types.InputContext) error {
	decoder, err := framestream.NewDecoder(r, i.DecoderOptions)
	if err != nil {
		i.fstrmDecordErrCount.Inc()
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
			i.fstrmDecordErrCount.Inc()
			return errors.Wrap(err, "failed to decode DNSTAP message")
		}
		dm, err := i.unmarshaler(bs)
		if err != nil {
			i.unmarshalDecordErrCount.Inc()
			return errors.Wrap(err, "failed to create dnstap message")
		}
		buf.Write(dm)
	}
	return nil
}

func init() {
	RegisterFormat(FormatMeta{Format: FormatDNSTAP}, NewDnstapInputServer)
	RegisterFormat(FormatMeta{Format: FormatDtapFrame}, NewDtapFrameInputServer)
}
