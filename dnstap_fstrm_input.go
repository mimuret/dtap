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
	"io"

	"github.com/pkg/errors"

	dnstap "github.com/dnstap/golang-dnstap"
	log "github.com/sirupsen/logrus"

	framestream "github.com/farsightsec/golang-framestream"
)

type DnstapFstrmInput struct {
	decoder  *framestream.Decoder
	rc       io.ReadCloser
	readDone chan struct{}
	finished bool
}

func NewDnstapFstrmInput(rc io.ReadCloser, bi bool) (*DnstapFstrmInput, error) {
	decoder, err := framestream.NewDecoder(rc, &framestream.DecoderOptions{
		ContentType:   dnstap.FSContentType,
		Bidirectional: bi,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "can't create framestream Decoder")
	}
	return &DnstapFstrmInput{
		rc:       rc,
		decoder:  decoder,
		readDone: make(chan struct{}),
	}, nil
}
func (i *DnstapFstrmInput) read(rbuf *RBuf, errCh chan error) {
	for {
		buf, err := i.decoder.Decode()
		if err != nil {
			if err == io.EOF {
				close(i.readDone)
				return
			}
			if log.GetLevel() >= log.DebugLevel {
				errCh <- errors.Wrapf(err, "fstrm decode error")
			}
			break
		}
		newbuf := make([]byte, len(buf))
		copy(newbuf, buf)
		rbuf.Write(newbuf)
	}
}
func (i *DnstapFstrmInput) Read(ctx context.Context, rbuf *RBuf, errCh chan error) {
	go i.read(rbuf, errCh)
	for {
		select {
		case <-ctx.Done():
			i.rc.Close()
		case <-i.ReadDone():
			break
		}
	}
}

func (i *DnstapFstrmInput) ReadDone() <-chan struct{} {
	return i.readDone
}
