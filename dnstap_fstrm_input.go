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

	framestream "github.com/farsightsec/golang-framestream"
)

type DnstapFstrmInput struct {
	decoder   *framestream.Decoder
	rc        io.ReadCloser
	readError chan error
	finished  bool
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
		rc:        rc,
		decoder:   decoder,
		readError: make(chan error),
	}, nil
}
func (i *DnstapFstrmInput) read(rbuf *RBuf) {
	for {
		buf, err := i.decoder.Decode()
		if err != nil {
			if err == io.EOF {
				i.readError <- nil
				return
			}
			i.readError <- errors.Wrap(err, "decode error")
			return
		}
		newbuf := make([]byte, len(buf))
		copy(newbuf, buf)
		rbuf.Write(newbuf)
	}
}
func (i *DnstapFstrmInput) Read(ctx context.Context, rbuf *RBuf) error {
	var err error
	go i.read(rbuf)
L:
	for {
		select {
		case <-ctx.Done():
			i.rc.Close()
		case err = <-i.readError:
			break L
		}
	}
	return err
}
