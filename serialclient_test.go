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
	"flag"
	"fmt"
	"testing"
	"time"
)

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
