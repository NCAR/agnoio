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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"regexp"
	"time"

	"testing"
)

func TestNewArbiter(t *testing.T) {
	if _, err := NewArbiter(context.Background(), 0, "no-op"); err == nil {
		t.Error("Invalid dial should return an error")
	}
}

func arbHandler(t *testing.T, con net.Conn) {
	t.Helper()
	defer con.Close()
	for {
		// fmt.Println("\t\tarbHandler>> Waiting for Bytes")
		buf := make([]byte, 1024)
		reqLen, err := con.Read(buf)
		switch err {
		case nil:
			// fmt.Printf("\t\tarbHandler>> Got %q - Writing 'Rxd>%d'\n", string(buf[0:reqLen]), reqLen)
			fmt.Fprintf(con, "Rxd>%d", reqLen)
		case io.EOF:
			// fmt.Println("\t\tarbHandler>> EOF")
			return
		default:
			// fmt.Println("\t\tarbHandler>> ", err)
			return
		}
	}
}

func TestArb(t *testing.T) {
	//startup TCP server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go newTCPSvr(ctx, t, "tcp", "localhost:5003", arbHandler)

	a, e := NewArbiter(ctx, 500*time.Millisecond, "tcp://localhost:5003")
	if a == nil || e != nil {
		t.Log("Arb: ", a)
		t.Log("Err: ", e)
		t.Error("Need to open arbiter to test...")
		t.FailNow()
	}

	_ = a.String()

	//Write a known message
	if e := a.Open(); e != nil {
		t.Error("Should have returned a nil error")
		t.FailNow()
	}

	//Write a known message
	if n, e := a.Write([]byte("Garbage")); n != 7 || e != nil {
		t.Log("Wrote:", n, "wanted 7")
		t.Log("Err:", e, "wanted nil")
		t.Error("Didnt write what I needed to")
		t.FailNow()
	}

	b := make([]byte, 128)
	if n, e := a.Read(b); n != 5 || e != nil {
		t.Log("Wrote:", n, "wanted 5")
		t.Log("Err:", e, "wanted nil")
		t.Error("Didnt write what I needed to")
		t.FailNow()
	}

	if e := a.Close(); e != nil {
		t.Error("Should have returned a nil error")
		t.FailNow()
	}
}

var arbCmdBad, arbCmdOk, arbCmdError, arbCmdTimeout = Command{
	Name:          "bad command",
	Timeout:       500 * time.Millisecond,
	Prototype:     "ABC",
	CommandRegexp: regexp.MustCompile("^Cat"),
	Response:      regexp.MustCompile("Rxd>3"),
	Error:         nil,
}, Command{
	Name:          "all good",
	Timeout:       500 * time.Millisecond,
	Prototype:     "ABC",
	CommandRegexp: regexp.MustCompile(".*"),
	Response:      regexp.MustCompile("Rxd>3"),
	Error:         nil,
}, Command{
	Name:          "error matches",
	Timeout:       500 * time.Millisecond,
	Prototype:     "ABC",
	CommandRegexp: regexp.MustCompile(".*"),
	Response:      regexp.MustCompile("^a"),
	Error:         regexp.MustCompile("Rxd>3"),
},
	Command{
		Name:          "timeout",
		Timeout:       500 * time.Millisecond,
		Prototype:     "ABC",
		CommandRegexp: regexp.MustCompile(".*"),
		Response:      regexp.MustCompile("^a"),
		Error:         nil,
	}

func TestArb_Control(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go newTCPSvr(ctx, t, "tcp", "localhost:5002", arbHandler)
	a, e := NewArbiter(ctx, 500*time.Millisecond, "tcp://localhost:5002")
	if e != nil {
		t.Error("Unable to dial", e)
	}
	defer a.Close()

	if resp := a.Control(arbCmdBad); resp.Error == nil {
		t.Error("Expected a broken command to fail")
		t.FailNow()
	}

	//populate the buffer by writing some garbage
	for i := 0; i < 10; i++ {
		if i, e := a.Write([]byte("dead cat bounce")); i != 15 || e != nil {
			t.Log("Wrote expecting 15 bytes out: only ", i)
			t.Log("Err is ", e)
			t.Error("Unabl to fill bufffer")
			t.FailNow()
		}
	}
	<-time.After(20 * time.Millisecond)

	if resp := a.Control(arbCmdOk); resp.Error != nil {
		t.Log("Got err", resp.Error)
		t.Log("Got Bytes", string(resp.Bytes))
		t.Error("Expected response to arb a command to respond with nil")
		t.FailNow()
	}

	if resp := a.Control(arbCmdTimeout); resp.Error == nil || !bytes.Equal(resp.Bytes, []byte("Rxd>3")) {
		t.Log("Got err", resp.Error)
		t.Log("Got Bytes", string(resp.Bytes))
		t.Error("Expected a non-nill error due to a timeout")
		t.FailNow()
	}

	if resp := a.Control(arbCmdError); resp.Error == nil || !bytes.Equal(resp.Bytes, []byte("Rxd>3")) {
		t.Log("Got err", resp.Error)
		t.Log("Got Bytes", string(resp.Bytes))
		t.Error("Expected a non-nill error due to a timeout")
		t.FailNow()
	}

}
