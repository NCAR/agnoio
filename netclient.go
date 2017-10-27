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
	"fmt"
	"net"
	"regexp"
	"time"
)

var _ IDoIO = &NetClient{}
var netClientRe = regexp.MustCompile("^(tcp|tcp4|tcp6|udp|udp4|udp6):\\/\\/(.*:[a-zA-Z0-9]*)$")

/*NewNetClient opens a connection to remote tcpv4 host.
dial should be in the form of: 'tcp|udp[46]{0,1}://<host>:<port>'

Timeout is used a read/write timeout at the socket level. If timeout is zero,
timeouts are not used nor applied, and any errors are due to normal socket behaviour.
If timeout is greater than zero, a deadline is set on every Read() and Write()
function. In this case, Read() and Write() will returns an timeout error that
should be checked via something like the following:

  io := NewNetClient(ctx, 100 * time.Millisecond, "tcp://localhost:4242")
  ...
  n, e := io.Write(b)
  switch e {
  case io.EOF: // Broken socket
    ...
  default:
    if nerr, ok := e.(net.Error); ok {
      if nerr.Temporary() { //Temporary error
        ...
      }
      if nerr.Timeout() { //Timeout error from enforced deadline
        ...
      }
    }
  }

The caller is responsible for handling errors. This pkg just propegates any error
encountered.

*/
func NewNetClient(ctx context.Context, timeout time.Duration, dial string) (*NetClient, error) {
	if !netClientRe.MatchString(dial) {
		return nil, newErr(false, false, fmt.Errorf("dial string not in correct form"))
	}
	matches := netClientRe.FindAllStringSubmatch(dial, -1) //capture groups used
	nctx, cancel := context.WithCancel(ctx)
	nc := &NetClient{
		network:   matches[0][1],
		address:   matches[0][2],
		timeout:   timeout,
		rwtimeout: 1 * time.Millisecond,
		ctx:       nctx,
		cancel:    cancel,
	}
	return nc, nc.Open()
}

/*NetClient provides a implementer of the  IOStreamer interface.  It provides
access under the following URI Regimes:
  tcp://
  tcp4://
  tcp6://
  udp://
  udp4://
  udp6://


*/
type NetClient struct {
	network, address string
	cancel           context.CancelFunc
	ctx              context.Context
	rwtimeout        time.Duration
	timeout          time.Duration
	conn             net.Conn
}

/*String conforms to the fmt.Stringer interface*/
func (nc *NetClient) String() string {
	return fmt.Sprintf("%v connection to %v", nc.network, nc.address)
}

/*Open forcible disconnects (ignore errors) the network conenction and
attempts the connect process again.  It returns an error if it was unable to start*/
func (nc *NetClient) Open() (err error) {
	select {
	case <-nc.ctx.Done():
		return newErr(false, false, nc.ctx.Err())
	default:
	}
	if nc.conn != nil {
		nc.conn.Close()
		nc.conn = nil
	}
	dialer := net.Dialer{
		Timeout: nc.timeout,
		// Deadline:
		// LocalAddr:
		// FallbackDelay:
		DualStack: false,
		KeepAlive: 1 * time.Second,
		Resolver:  nil,
	}
	//Errors from DialContext implent net.Error
	nc.conn, err = dialer.DialContext(nc.ctx, nc.network, nc.address)
	return
}

/*Read conforms to io.Writer, but immediately returns upon ctx
destruction after closing the underling transport*/
func (nc *NetClient) Read(b []byte) (int, error) {
	select {
	case <-nc.ctx.Done():
		defer nc.Close()
		return 0, newErr(false, false, nc.ctx.Err())
	default:
		if nc.rwtimeout > 0 {
			nc.conn.SetReadDeadline(time.Now().Add(nc.rwtimeout))
		}
		return nc.conn.Read(b) //nc.conn  return errors that conform to net.Error
	}
}

/*Write conforms to io.Writer, but immediately returns upon ctx
destruction after closing the underling transport*/
func (nc *NetClient) Write(b []byte) (int, error) {
	select {
	case <-nc.ctx.Done():
		defer nc.Close()
		return 0, newErr(false, false, nc.ctx.Err())
	default:
		if nc.rwtimeout > 0 {
			nc.conn.SetWriteDeadline(time.Now().Add(nc.rwtimeout))
		}
		return nc.conn.Write(b) //nc.conn  return errors that conform to net.Error
	}
}

/*Close conforms to io.Closer, but immediately returns upon ctx
destruction after closing the underling transport*/
func (nc *NetClient) Close() error {
	nc.cancel()
	defer func() { nc.conn = nil }()
	if nc.conn != nil {
		return nc.conn.Close()
	}
	return nil

	// select {
	// case <-nc.ctx.Done():
	// 	return newErr(false, false, nc.ctx.Err()) //Context closed: return that error
	// default:
	// 	if nc.conn != nil {
	// 		return nc.conn.Close()
	// 	}
	// 	return nil
	// }
}
