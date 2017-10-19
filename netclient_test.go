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
	"math/rand"
	"net"
	"testing"
	"time"
)

type respHandler func(*testing.T, net.Conn)

func echoHandler(t *testing.T, con net.Conn) {
	t.Helper()
	defer con.Close()
	for {
		buf := make([]byte, 1024)
		reqLen, err := con.Read(buf)
		if err != nil {
			t.Log("Echo> ", err.Error())
			return
		}
		con.Write(buf[0:reqLen])
	}
}

func randPortCfg() (port int, svr string, dial string) {
	rand.Seed(time.Now().UnixNano())
	port = rand.Intn(4000) + 2000
	svr = fmt.Sprintf("localhost:%d", port)
	dial = fmt.Sprintf("tcp://localhost:%d", port)
	return
}

func newTCPSvr(ctx context.Context, t *testing.T, proto string, addr string, handler respHandler) {
	t.Helper()
	svr, err := net.Listen(proto, addr)

	if err != nil {
		t.Error(err)
		t.Error("Unable to start server")
		panic(err)
	}
	t.Log("Listening on ", proto, addr)
	go func() {
		defer svr.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			con, err := svr.Accept()
			if err != nil {
				t.Log("Connection Error:", err)
			}
			go handler(t, con)
			defer con.Close()
		}
	}()
}

func TestNewNetClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := NewIDoIO(ctx, 1*time.Millisecond, "bad hair day"); err == nil {
		t.Error("Bad dial string should fail")
		t.FailNow()
	}
	if _, err := NewNetClient(ctx, 1*time.Millisecond, "tcp://bad-hair-day"); err == nil {
		t.Error("Bad dial string should fail")
		t.FailNow()
	}
	port, svrdial, dial := randPortCfg()
	t.Logf("Starting server on port %d", port)
	newTCPSvr(ctx, t, "tcp4", svrdial, echoHandler)

	nc, err := NewIDoIO(ctx, 1*time.Millisecond, dial)
	_ = nc.String()
	if err != nil {
		t.Error("Shouldnt get an error")
		t.FailNow()
	}

	//Write some garbage
	msg := []byte("a dead cow sings the blues")
	if n, e := nc.Write(msg); e != nil || n != len(msg) {
		t.Log("Wanted to write", len(msg), "bytes, wrote", n)
		t.Log("Error was ", e)
		t.Error("Write is borked")
		t.FailNow()
	}

	read := make([]byte, 1024)
	if n, e := nc.Read(read); e != nil || n != len(msg) {
		t.Log("Wanted to read", len(msg), "bytes, only read", n)
		t.Log("Error was ", e)
		t.Error("Read is borked")
		t.FailNow()
	}

	for i := 0; i < 10; i++ {
		nc.Close()
	}
	cancel() //kill context - expecting nothing but errors from here

	//Write some garbage
	if n, e := nc.Write(msg); e == nil || n != 0 {
		t.Log("Wanted to write 0 bytes, wrote", n)
		t.Log("Error was nil")
		t.Error("Write is borked")
		t.FailNow()
	}

	if n, e := nc.Read(read); e == nil || n != 0 {
		t.Log("Wanted to read 0 bytes, read", n)
		t.Log("Error was nil")
		t.Error("Read is borked")
		t.FailNow()
	}
	//attempt reopen on dead context

	if err := nc.Open(); err == nil {
		t.Error("Should always get an error on a dead context")
		t.FailNow()
	}
}
