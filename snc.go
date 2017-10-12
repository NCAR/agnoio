//+build ignore

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

package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"

	"github.com/NCAR/agnoio"
	"github.com/alecthomas/kingpin"
)

var (
	app  = kingpin.New("snc", "A crappy netcat with fewer options, but can talk serial")
	dial = app.Arg("dial", "Dial string").Default("tcp://localhost:2000").String()
)

func main() {
	_ = kingpin.MustParse(app.Parse(os.Args[1:]))
	con, err := agnoio.Create(context.Background(), 1*time.Second, *dial)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			b := make([]byte, 1024)
			if n, e := con.Read(b); e != io.EOF {
				os.Stdout.Write(b[0:n])
			}
		}
	}()

	//read from stdin
	stdin := bufio.NewReader(os.Stdin)
	for {
		if line, err := stdin.ReadSlice('\n'); err == nil {
			con.Write(line)
		}
	}
}

/*



 */
