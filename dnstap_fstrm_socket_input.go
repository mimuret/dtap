/*
 * Copyright (c) 2018 Manabu Sonoda
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dtap

import (
	"context"
	"net"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var closeWant string = "use of closed network connection"

type DnstapFstrmSocketInput struct {
	listener  net.Listener
	readDone  chan struct{}
	readError chan error
}

func NewDnstapFstrmSocketInput(listener net.Listener) (*DnstapFstrmSocketInput, error) {
	return &DnstapFstrmSocketInput{
		listener:  listener,
		readError: make(chan error),
		readDone:  make(chan struct{}),
	}, nil
}

func (i *DnstapFstrmSocketInput) runRead(ctx context.Context, rbuf *RBuf) {
	readCtx, readCancel := context.WithCancel(ctx)
	for {
		conn, err := i.listener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), closeWant) && log.GetLevel() >= log.InfoLevel {
				i.readError <- errors.Wrapf(err, "can't accept unix socket")
				return
			}
			readCancel()
			close(i.readDone)
			break
		}
		input, err := NewDnstapFstrmInput(conn, true)
		if err != nil {
			if log.GetLevel() >= log.InfoLevel {
				readCancel()
				close(i.readDone)
				i.readError <- errors.Wrapf(err, "can't create NewDnstapFstrmInput")
				return
			}
		}
		childCtx, _ := context.WithCancel(readCtx)
		go input.Read(childCtx, rbuf)
	}
	return
}

func (i *DnstapFstrmSocketInput) Run(ctx context.Context, rbuf *RBuf) error {
	var err error
	childCtx, _ := context.WithCancel(ctx)
	go i.runRead(childCtx, rbuf)
	select {
	case <-ctx.Done():
		i.listener.Close()
	case err = <-i.readError:
		break
	case <-i.ReadDone():
		break
	}
	log.Info("finish input")
	return err
}

func (i *DnstapFstrmSocketInput) ReadDone() <-chan struct{} {
	return i.readDone
}
