package input

import (
	"io"
	"net"
	"sync"

	dnstap "github.com/dnstap/golang-dnstap"
	framestream "github.com/farsightsec/golang-framestream"
	"github.com/mimuret/dtap/v2/pkg/logger"
	"github.com/mimuret/dtap/v2/pkg/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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

type InputServer struct {
	DecoderOptions    *framestream.DecoderOptions
	logger            *zap.Logger
	connectionManager *connectionManager
}

func NewInputServer(options *framestream.DecoderOptions) *InputServer {
	if options == nil {
		options = &framestream.DecoderOptions{
			ContentType:   dnstap.FSContentType,
			Bidirectional: true,
		}
	}
	return &InputServer{
		DecoderOptions:    options,
		logger:            logger.GetLogger(),
		connectionManager: newConnectionManager(),
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
				i.logger.Error("input error", zap.Error(err))
			}
			i.connectionManager.remove(conn)
			wg.Done()
		}(conn)
	}
}

func (i *InputServer) Read(r io.Reader, buf types.Writer) error {
	decoder, err := framestream.NewDecoder(r, i.DecoderOptions)
	if err != nil {
		return err
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
		dm, err := types.NewDnstapMessage(bs)
		if err != nil {
			return errors.Wrap(err, "failed to create dnstap message")
		}
		buf.Write(dm)
	}
	return nil
}
