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

/*ExitCriteria is a set of defined success criteria that CheckFunc must return*/
type ExitCriteria int

const (
	//Insufficient should be returned if more bytes are required in order to determine success or failure
	Insufficient ExitCriteria = 1 + iota

	//Failure indicates the current set of bytes indicates an error condition
	Failure

	//Success indicates the current set of bytes indicates an accepted condition.
	Success
)

/*CheckFunc is used to determine if the passed bytes match some success, failure,
or insuccifient data to determine exit criteria. Only the defined ExitCriteria
may be used - any other return value will panic.  If the CheckFunc returns
Insufficient, it is assumed that more incoming data is required before a success
or failure criteria can be established. If Failure is returned, it is assumed
that the sum of the bytes demarcates a failure condition, and the calling process
should cease reading data.  Likewise, a Success condition indicates a successful
exit criteria, and the calling process should cease reading data and return a nil
error.*/
type CheckFunc func([]byte) ExitCriteria

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

	/*Simple is a very simple form of command and control.  It sends out cmd,
	making sure all the bytes get pushed out, and the constantly reads the incoming
	data for any a sequence that contains either 'ok' or 'failure' before timing
	out at the passed duration. The returned response contains the duration,
	the bytes received, and an error, which is nil if the ok sequence was
	detected, or a non-nil error*/
	Simple(cmd, ok, failure []byte, duration time.Duration) Response

	/*Control forms a byte slice to write out on the wire by combining cmd with
	args, and sans error, will write the formed byte slice out on the wire. It
	should block until either its internal buffer matches cmd.Response, cmd.Error,
	or the process takes longer than cmd.Timeout. The returned Response should be
	populated correctly as described in the Response docstring*/
	Control(cmd Command, args ...interface{}) Response
}

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

/*clearBuffer removes items from the internal buffers*/
func (a *Arb) clearBuffer() {
	//clear off any internal buffer
	rdr := bufio.NewReader(a.idotoo)
	for {
		_, e := rdr.ReadByte()
		if e != nil {
			break
		}
	}
}

/*Simple is a very dumb version of control IO*/
func (a *Arb) Simple(cmd, success, failure []byte, duration time.Duration) (rsp Response) {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.clearBuffer()
	start := time.Now()
	defer func() { rsp.Duration = time.Since(start) }()

	//send off the bytes, barfing on any sort of write error
	if n, werr := a.idotoo.Write(cmd); werr != nil || len(cmd) != n {
		return Response{Error: fmt.Errorf("unable to write full message of %d bytes (wrote %d) withot an error: %v", len(cmd), n, werr)}
	}

	//creating data channel for communicating with reader
	dataChan := make(chan status, 0)

	cf := func(raw []byte) ExitCriteria {
		if failure != nil && bytes.Contains(raw, failure) {
			return Failure
		}
		if success != nil && bytes.Contains(raw, success) {
			return Success
		}
		return Insufficient
	}

	// part of the contact of readUntil is that we must read from the passed channel.
	// It will write the necessary data if the ctx collapses.
	go a.readUntil(dataChan, duration, cf)
	d := <-dataChan
	return Response{Error: d.err, Bytes: d.raw}
}

/*Control conforms to Arbiter interface, but this implementation uses a IDoIO to
handles the data. Control is the reason that serialzed access is required:  when
Commands are sent, Control needs to read all the incoming data while Checking
for a valid Response.

If .CommandRegexp is nil, whatever command is formed is not checked for
completeness (see Command.Bytes) If .Error is nil (not set), then the output is
not compared for an error condition, and the command will only succeed or
timeout. If .Response is nil (not set), then the output is not compared for an
positive repsonse, and Command will only fail or timeout.  If bother .Error and
.Response are nil, this command will only timeout.
*/
func (a *Arb) Control(cmd Command, args ...interface{}) (rsp Response) {
	//Any sort of formatting error gets kicked back immediately
	rawBytes, err := cmd.Bytes(args...)
	if err != nil {
		return Response{Error: err}
	}

	a.mux.Lock()
	defer a.mux.Unlock()

	a.clearBuffer()
	//send off the bytes, barfing on any sort of write error
	if n, werr := a.idotoo.Write(rawBytes); werr != nil || len(rawBytes) != n {
		return Response{Error: fmt.Errorf("unable to write full message of %d bytes (wrote %d) withot an error: %v", len(rawBytes), n, werr)}
	}

	start := time.Now()
	defer func() { rsp.Duration = time.Since(start) }()

	//creating data channel for communicating with reader
	dataChan := make(chan status, 0)

	cf := func(raw []byte) ExitCriteria {
		if cmd.Error != nil && cmd.Error.Match(raw) { //check for error response
			return Failure
		}
		if cmd.Response != nil && cmd.Response.Match(raw) { //check for normal acceptable response
			return Success
		}
		return Insufficient
	}

	// part of the contact of readUntil is that we must read from the passed channel.
	// It will write the necessary data if the ctx collapses.
	go a.readUntil(dataChan, cmd.Timeout, cf)
	d := <-dataChan
	return Response{Error: d.err, Bytes: d.raw}
}

/*status is used to pass messages from readUntil back to callers.*/
type status struct {
	raw []byte
	err error
}

/*readUntil repeatedly reads data off the embedded io device until either a
duration of timeout elapses, or checkFunc returns either Success or Failure.
Caller should utilize a go-routine to issue this and should always read from
the passed channel exactly one time, otherwise this will deadlock. This closes
the channel on exit.
*/
func (a *Arb) readUntil(dataChan chan<- status, timeout time.Duration, checkFunc CheckFunc) {
	timeoutctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer close(dataChan)
	defer cancel()
	rcvd, buf := bytes.NewBuffer(nil), bufio.NewReader(a.idotoo)

	for {
		select {
		case <-a.ctx.Done(): //context chain has collapsed
			dataChan <- status{err: errors.Wrap(a.ctx.Err(), "Arbiter's context chain has collapsed"), raw: rcvd.Bytes()}
			return
		case <-timeoutctx.Done(): //timeout
			dataChan <- status{err: errors.Wrap(timeoutctx.Err(), "Command timed out before receiving the proper response"), raw: rcvd.Bytes()}
			return
		default:
		}

		for {
			b, e := buf.ReadByte()
			if e != nil {
				break
			}
			rcvd.WriteByte(b)
		}

		raw := rcvd.Bytes()
		switch checkFunc(raw) {
		case Insufficient: //need more data
		case Failure: //return failure
			dataChan <- status{err: errors.New("Command received error response"), raw: raw}
			return
		case Success:
			dataChan <- status{err: nil, raw: raw}
		}
	}
}
