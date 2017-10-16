package agnoio

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

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
)

/*
Arbiter provides a command and control interface to []byte streams. Original
design intentions were to provide a way to communicate to devices that respond
to 'commands' sent over the wire. Functionally, this can be seen as a socket or
generic IO wrapper to provide a way to read and write commands and data. As a
sanity, there can only be one caller, as this is purposefully not safe from
mutliple callers via the standard "go <func>" syntax.  Any errors that are not
ErrTimeout or ErrBusy are errors coming from the underlying layers and are to
be delt with*/
type Arbiter interface {
	IDoIO //I Do Too

	/*Control forms a byte slice to write out on the wire by combining cmd with
	args, and sans error, will write the formed byte slice out on the wire. It
	should block until either its internal buffer matches cmd.Response, cmd.Error,
	or the process takes longer than cmd.Timeout. The returned Response should be
	populated correctly as described in the Response docstring*/
	Control(cmd Command, args ...interface{}) Response
}

//arbitrate creates and Arbiter out of a IDoIO
// func arbitrate(idoio IDoIO, err error) (Arbiter, error) {
// 	return &arb{idotoo: idoio}, err
// }

/*NewArbiter returns an opened Arbiter from the passed dial string, ctx, and timeout.
dial will need to match a known dial format, timeout will be used during the connection
process, and the ctx will be used to ensure the operation will cease if the ctx is
stopped.*/
func NewArbiter(ctx context.Context, timeout time.Duration, dial string) (Arbiter, error) {
	idotoo, err := NewIDoIO(ctx, timeout, dial)
	arb, _ := Arbitrate(ctx, idotoo)
	return arb, err
}

/*Arbitrate returns an Arbiter and a context.CancelFunc.  This is meant to be a
temporary solution, where the arbiter is meant to be used for a short duration
and then revert to using the IDoIO. The CancelFunc should be called whenever
the caller is done using the Arbiter Functionally (eg, .Control).*/
func Arbitrate(ctx context.Context, idoio IDoIO) (Arbiter, context.CancelFunc) {
	arbctx, cancelfunc := context.WithCancel(ctx)
	return &Arb{ctx: arbctx, idotoo: idoio, cancel: cancelfunc}, cancelfunc
}

/*Arb is a wrapper over a IDoIO, but it lockes the IDoIO under a mutex to
serialize access.*/
type Arb struct {
	ctx    context.Context
	cancel context.CancelFunc
	mux    sync.Mutex //only one reader and writer: me
	idotoo IDoIO
}

/*String  conforms to IDoIO, but for an Arbiter.  Unlike a regular IDoIO, access is
locked within a mutex, and the read and write channels are linked*/
func (a *Arb) String() string {
	return fmt.Sprintf("Arbiter over %s", a.idotoo.String())
}

/*Open conforms to IDoIO, but for an Arbiter.  Unlike a regular IDoIO, access is
locked within a mutex, and the read and write channels are linked*/
func (a *Arb) Open() error {
	a.mux.Lock()
	defer a.mux.Unlock()
	return a.idotoo.Open()
}

/*Close conforms to IDoIO and io.Closer, but for an Arbiter. Unlike a regular
IDoIO, access is locked within a mutex, and the read and write channels are linked*/
func (a *Arb) Close() error {
	a.mux.Lock()
	defer a.mux.Unlock()
	return a.idotoo.Open()
}

/*Read conforms to IDoIO, io.Reader, but for an Arbiter. Unlike a regular IDoIO,
access is locked within a mutex, and the read and write channels are linked*/
func (a *Arb) Read(b []byte) (int, error) {
	a.mux.Lock()
	defer a.mux.Unlock()
	return a.idotoo.Read(b)
}

/*Write conforms to IDoIO, io.Writer, but for an Arbiter. Unlike a regular IDoIO,
access is locked within a mutex, and the read and write channels are linked*/
func (a *Arb) Write(b []byte) (int, error) {
	a.mux.Lock()
	defer a.mux.Unlock()
	return a.idotoo.Write(b)
}

/*Control conforms to Arbiter interface, but this implementation uses a IDoIO to
handles the data. Control is the reason that serialzed access is required:  when
Commands are sent, Control needs to read all the incoming data while Checking
for a valid Response.  Generally, the steps are something like the following:

1) Render bytes
*/
func (a *Arb) Control(cmd Command, args ...interface{}) (rsp Response) {
	a.mux.Lock()
	defer a.mux.Unlock()
	start := time.Now()
	defer func() { rsp.Duration = time.Since(start) }()

	//Any sort of formatting error gets kicked back immediately
	rawBytes, err := cmd.Bytes(args...)
	if err != nil {
		return Response{Error: err}
	}

	//clear off any internal buffer
	rdr := bufio.NewReader(a.idotoo)
	for {
		_, e := rdr.ReadByte()
		if e != nil {
			break
		}
	}

	//send off the bytes, barfing on any sort of write error
	if n, werr := a.idotoo.Write(rawBytes); werr != nil || len(rawBytes) != n {
		return Response{Error: fmt.Errorf("unable to write full message of %d bytes (wrote %d) withot an error: %v", len(rawBytes), n, werr)}
	}

	//creating data channel for communicating with reader
	dataChan := make(chan cr, 0)
	go a.waitForResponse(dataChan, cmd)

	select { //block until our context is killed, or we get a response
	case <-a.ctx.Done():
		return Response{Error: a.ctx.Err()}
	case respData := <-dataChan:
		return Response{Error: respData.e, Bytes: respData.b}
	}
}

/*cr is used to pass messages between Control and waitForResponse*/
type cr struct {
	b []byte
	e error
}

/*waitForResponse repeatedly reads from the IDoIO waiting for a response, and */
func (a *Arb) waitForResponse(dataChan chan<- cr, cmd Command) {
	timeoutctx, cancel := context.WithTimeout(a.ctx, cmd.Timeout)
	defer close(dataChan)
	defer cancel()
	rcvd, buf := bytes.NewBuffer(nil), bufio.NewReader(a.idotoo)

	for {
		select {
		case <-a.ctx.Done(): //context chain has collapsed
			dataChan <- cr{e: errors.Wrap(a.ctx.Err(), "Arbiter's context chain has collapsed"), b: rcvd.Bytes()}
			return
		case <-timeoutctx.Done(): //timeout
			dataChan <- cr{e: errors.Wrap(timeoutctx.Err(), "Command timed out before receiving the proper response"), b: rcvd.Bytes()}
			return
		case <-time.After(1 * time.Millisecond):
			// default:
		}

		for {
			b, e := buf.ReadByte()
			if e == nil {
				rcvd.WriteByte(b)
			} else {
				break
			}
		}

		raw := rcvd.Bytes()
		if cmd.Error != nil && cmd.Error.Match(raw) { //check for error response
			dataChan <- cr{e: errors.New("Command received error response"), b: raw}
			return
		}

		if cmd.Response.Match(raw) { //check for normal acceptable response
			dataChan <- cr{e: nil, b: raw}
			return
		}
	}
}

/*





 */
