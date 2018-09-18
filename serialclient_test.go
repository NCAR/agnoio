/*
MIT License

Copyright (c) 2015-2018 University Corporation for Atmospheric Research

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
	"flag"
	"fmt"
	"io"
	"testing"
	"time"

	"go.bug.st/serial.v1"
)

type tstport struct {
	read, write func([]byte) (int, error)
	close       func() error
}

func (tp *tstport) SetMode(*serial.Mode) error { return nil }
func (tp *tstport) Read(p []byte) (int, error) {
	if tp.read != nil {
		return tp.read(p)
	}
	return 0, nil
}
func (tp *tstport) Write(p []byte) (int, error) {
	if tp.write != nil {
		return tp.write(p)
	}
	return 0, nil
}

func (tp *tstport) ResetInputBuffer() error  { return nil }
func (tp *tstport) ResetOutputBuffer() error { return nil }
func (tp *tstport) SetDTR(dtr bool) error    { return nil }
func (tp *tstport) SetRTS(rts bool) error    { return nil }
func (tp *tstport) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &serial.ModemStatusBits{}, nil
}
func (tp *tstport) Close() error {
	if tp.close != nil {
		return tp.close()
	}
	return nil
}

var _ = serial.Port(&tstport{})

var (
	port = flag.String("port", "", "Serial port to use as a loopback test")
)

func TestMain(t *testing.T) {
	flag.Parse()
}

func TestNewSerialClient(t *testing.T) {
	if *port == "" {
		t.Skip("No serial port defined for loopback tests - skipping")
		t.SkipNow()
	}
	dial := fmt.Sprintf("serial://%s:57600", *port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := NewIDoIO(ctx, 0, "bad hair day"); err == nil {
		t.Error("Bad dial string should fail")
	}
	if _, err := NewSerialClient(ctx, 0, "tcp://bad-hair-day:012301231201230123w31203012301030"); err == nil {
		t.Error("Bad dial string should fail")
	}

	sc, err := NewIDoIO(ctx, 0, dial)
	if err != nil {
		t.Error("Shouldnt get an error", err)
	}
	_ = sc.String()

	// Write some garbage
	msg := []byte("a dead cow sings the blues")
	if n, e := sc.Write(msg); e != nil || n != len(msg) {
		t.Log("Wanted to write", len(msg), "bytes, wrote", n)
		t.Log("Error was ", e)
		t.Error("Write is borked")
	}

	//serial actually takes measurable time. ~ # chars  * 1000 / baud Milliseconds
	<-time.After(time.Duration(len(msg)*1000/57600) * time.Millisecond)

	read := make([]byte, 1024)
	if n, e := sc.Read(read); e != nil || n != len(msg) {
		t.Log("Wanted to read", len(msg), "bytes, only read", n)
		t.Log("Error was ", e)
		t.Error("Read is borked")
	}

	for i := 0; i < 10; i++ {
		sc.Close()
	}
	cancel() //kill context - expecting nothing but errors from here

	//Write some garbage
	if n, e := sc.Write(msg); e == nil || n != 0 {
		t.Log("Wanted to write 0 bytes, wrote", n)
		t.Log("Error was nil")
		t.Error("Write is borked")
	}

	if n, e := sc.Read(read); e == nil || n != 0 {
		t.Log("Wanted to read 0 bytes, read", n)
		t.Log("Error was nil")
		t.Error("Read is borked")
	}
	//attempt reopen on dead context

	if err := sc.Open(); err == nil {
		t.Error("Should always get an error on a dead context")
	}
}

/*This tests a large chunk of the context failures without needing a serial port*/
func Test_SerialClient_NoConnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	if _, e := NewSerialClient(ctx, 0, "does-not-match-regexp"); e == nil {
		t.Error("Expected an error - shouldnt get a serial obj")
	}
	sc, err := NewSerialClient(ctx, 100, "serial://dontexit:57600")
	if err == nil {
		t.Error("Expected an error")
	}
	sc.conn = nil
	_ = sc.String()

	b := make([]byte, 16)
	if n, e := sc.Read(b); n != 0 || e == nil {
		t.Error("Reading a closed port should give some sort of error")
	}

	sc.conn = nil
	if e := sc.Open(); e == nil {
		t.Error("Reading a closed port should give some sort of error")
	}

	sc.conn = nil
	if n, e := sc.Write(b); n != 0 || e == nil {
		t.Error("Reading a closed port should give some sort of error")
	}
	sc.conn = nil
	conn := &tstport{close: func() error { return fmt.Errorf("cannot close close port") }}
	sc.conn = conn
	if e := sc.Close(); e == nil {
		t.Error("Closing a closed port should give some sort of error")
	}
	//here, and only here, close should return nil
	if e := sc.Close(); e != nil {
		t.Error("Closing here should give a nil error")
	}

	//Do some more  reads/writes/closes with a nil - shouldnt matter nor panic
	sc.conn = nil
	if n, e := sc.Read(b); n != 0 || e == nil {
		t.Error("Reading a closed port should give some sort of error")
	}
	if n, e := sc.Write(b); n != 0 || e == nil {
		t.Error("Reading a closed port should give some sort of error")
	}
	if e := sc.Close(); e != nil {
		t.Error("Closing a dead port should give a nil error")
	}

	//murder context, do it all again for the same response
	cancel()

	if n, e := sc.Read(b); n != 0 || e == nil {
		t.Error("Reading a closed port should give some sort of error")
	}
	sc.conn = nil

	if n, e := sc.Write(b); n != 0 || e == nil {
		t.Error("Reading a closed port should give some sort of error")
	}
	sc.conn = nil

	if e := sc.Close(); e == nil {
		t.Error("Closing a closed port should give some sort of error")
	}
}

func TestSerial_Open(t *testing.T) {
	ctx, cncl := context.WithCancel(context.Background())
	defer cncl()
	sc := &tstport{
		read:  func([]byte) (int, error) { return 0, nil },
		write: func([]byte) (int, error) { return 0, nil },
		close: func() error { return nil },
	}
	ser := &SerialClient{
		ctx:  ctx,
		conn: sc,
		mode: &serial.Mode{},
		dev:  "nope",
	}
	if err := ser.Open(); err == nil {
		t.Errorf("Expected an error - no such port")
	}

	cncl()
	if err := ser.Open(); err == nil {
		t.Errorf("Context dead, expect an error")
	}
}

func TestSerial_Read(t *testing.T) {
	type x struct {
		n    int
		conn serial.Port
		ctx  func() context.Context
	}

	tests := map[string]x{
		"canceled context": x{
			ctx: func() context.Context {
				ctx, cncl := context.WithCancel(context.Background())
				cncl()
				return ctx
			},
		},
		"nil connection": x{
			ctx:  context.Background,
			conn: nil,
		},
		"read 5 bytes with EOF": x{
			ctx:  context.Background,
			conn: &tstport{read: func([]byte) (int, error) { return 5, io.EOF }},
			n:    5,
		},
		"read 6 bytes with some other error": x{
			ctx:  context.Background,
			conn: &tstport{read: func([]byte) (int, error) { return 6, io.ErrClosedPipe }},
			n:    6,
		},
		"read 10 bytes with nil error": x{
			ctx:  context.Background,
			conn: &tstport{read: func([]byte) (int, error) { return 10, nil }},
			n:    10,
		},
	}

	for name, x := range tests {
		t.Log(name)
		ctx, cncl := context.WithCancel(x.ctx())
		defer cncl()
		ser := &SerialClient{
			ctx:    ctx,
			cancel: cncl,
			conn:   x.conn,
			mode:   &serial.Mode{},
			dev:    "nope",
		}
		if n, e := ser.Read([]byte{}); n != x.n {
			t.Log("err = ", e)
			t.Errorf("Expected %d bytes got %d", x.n, n)
		}
		cncl()
	}
}

func TestSerial_Write(t *testing.T) {
	type x struct {
		n    int
		conn serial.Port
		ctx  func() context.Context
	}

	tests := map[string]x{
		"canceled context": x{
			ctx: func() context.Context {
				ctx, cncl := context.WithCancel(context.Background())
				cncl()
				return ctx
			},
		},
		"nil connection": x{
			ctx:  context.Background,
			conn: nil,
		},
		"write 5 bytes with EOF": x{
			ctx:  context.Background,
			conn: &tstport{write: func([]byte) (int, error) { return 5, io.EOF }},
			n:    5,
		},
		"write 6 bytes with some other error": x{
			ctx:  context.Background,
			conn: &tstport{write: func([]byte) (int, error) { return 6, io.ErrClosedPipe }},
			n:    6,
		},
		"write 10 bytes with nil error": x{
			ctx:  context.Background,
			conn: &tstport{write: func([]byte) (int, error) { return 10, nil }},
			n:    10,
		},
	}

	for name, x := range tests {
		t.Log(name)
		ctx, cncl := context.WithCancel(x.ctx())
		defer cncl()
		ser := &SerialClient{
			ctx:    ctx,
			cancel: cncl,
			conn:   x.conn,
			mode:   &serial.Mode{},
			dev:    "nope",
		}
		if n, e := ser.Write([]byte{}); n != x.n {
			t.Log("err = ", e)
			t.Errorf("Expected %d bytes got %d", x.n, n)
		}
		cncl()
	}
}

func TestSerial_Close(t *testing.T) {
	type x struct {
		conn serial.Port
		ctx  func() context.Context
	}

	tests := map[string]x{
		"canceled context": x{
			ctx: func() context.Context {
				ctx, cncl := context.WithCancel(context.Background())
				cncl()
				return ctx
			},
		},
		"nil connection": x{
			ctx:  context.Background,
			conn: nil,
		},
		"close gracefully": x{
			ctx:  context.Background,
			conn: &tstport{},
		},
	}

	for name, x := range tests {
		t.Log(name)
		ctx, cncl := context.WithCancel(x.ctx())
		defer cncl()
		ser := &SerialClient{
			ctx:    ctx,
			cancel: cncl,
			conn:   x.conn,
			mode:   &serial.Mode{},
			dev:    "nope",
		}
		if ser.Close(); ser.conn != nil {
			t.Errorf("Despite errors, expected a nil sc.conn after")
		}
		cncl()
	}
}
