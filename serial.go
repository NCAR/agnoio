/*
MIT License

Copyright (c) 2015-2017 University Corporation for Atmospheric Research

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package agnoio

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/tarm/serial"
)

var _ IDoIO = &SerialClient{}
var serialRe = regexp.MustCompile("^rs232|serial:\\/\\/([^:]*):([0-9]*)$")

/*NewSerialClient opens a connection to a serial device in 8N1 mode.
Dial should be in the form of "serial://<device>:<baud>*/
func NewSerialClient(ctx context.Context, timeout time.Duration, dial string) (*SerialClient, error) {
	if !serialRe.MatchString(dial) {
		return nil, fmt.Errorf("dial string not in correct form")
	}
	matches := serialRe.FindAllStringSubmatch(dial, -1) //capture groups used
	i, _ := strconv.ParseInt(matches[0][2], 10, 64)
	nctx, cancel := context.WithCancel(ctx)

	sc := &SerialClient{
		ctx:     nctx,
		cancel:  cancel,
		timeout: timeout,
		opts: &serial.Config{
			Name:        matches[0][1],
			Baud:        int(i),
			ReadTimeout: timeout,
			Size:        8,
			Parity:      serial.ParityNone,
			StopBits:    serial.Stop1,
		},
		conn: &serial.Port{},
	}
	return sc, sc.Open()
}

/*SerialClient wraps around a serial port*/
type SerialClient struct {
	ctx     context.Context
	cancel  context.CancelFunc
	timeout time.Duration
	opts    *serial.Config
	conn    *serial.Port
}

/*String conforms to the fmt.Stringer interface*/
func (sc *SerialClient) String() string {
	return fmt.Sprintf("serial connection to %v:%d 8N1", sc.opts.Name, sc.opts.Baud)
}

/*Open forcible disconnects (ignore errors) the network conenction and
attempts the connect process again.  It returns an error if it was unable to start*/
func (sc *SerialClient) Open() (err error) {
	select {
	case <-sc.ctx.Done():
		return sc.ctx.Err()
	default:
	}
	if sc.conn != nil {
		sc.conn.Close()
		sc.conn = nil
	}
	sc.conn, err = serial.OpenPort(sc.opts)
	return
}

/*Read conforms to io.Writer, but immediately returns upon ctx
destruction after closing the underling transport*/
func (sc *SerialClient) Read(b []byte) (int, error) {
	select {
	case <-sc.ctx.Done():
		defer sc.Close()
		return 0, sc.ctx.Err()
	default:
		if sc.conn != nil {
			return sc.conn.Read(b)
		}
		return 0, errors.New("broken connection")
	}
}

/*Write conforms to io.Writer, but immediately returns upon ctx
destruction after closing the underling transport*/
func (sc *SerialClient) Write(b []byte) (int, error) {
	select {
	case <-sc.ctx.Done():
		defer sc.Close()
		return 0, sc.ctx.Err()
	default:
		if sc.conn != nil {
			return sc.conn.Write(b)
		}
		return 0, errors.New("broken connection")

	}
}

/*Close conforms to io.Closer, but immediately returns upon ctx
destruction after closing the underling transport*/
func (sc *SerialClient) Close() error {
	defer func() { sc.conn = nil }()
	select {
	case <-sc.ctx.Done():
		return sc.ctx.Err() //Context closed: return that error
	default:
		if sc.conn != nil {
			return sc.conn.Close()
		}
		return nil
	}
}
