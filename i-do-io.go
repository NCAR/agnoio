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
	"context"
	"fmt"
	"io"
	"regexp"
	"time"
)

/*IDoIO is a generic interface agnostic io devices should conforms to. An IDoIO
should be able to tell others in some human readable string form what the
transport actually is (fmt.Stringer). An IDoIO should alow be able to read,
and write byte slices (io.ReadWriter), and also should be able to Open and Close
the device as will.  This does mean that once created, an IDoIO needs to cache
and properly deal with opening critera.

Any error returned must be castable to net.Error*/
type IDoIO interface {
	fmt.Stringer
	io.ReadWriter
	io.Closer
	Open() error
}

var known = map[*regexp.Regexp]func(context.Context, time.Duration, string) (IDoIO, error){
	netClientRe: func(ctx context.Context, dur time.Duration, dial string) (IDoIO, error) {
		return NewNetClient(ctx, dur, dial)
	},
	serialRe: func(ctx context.Context, dur time.Duration, dial string) (IDoIO, error) {
		return NewSerialClient(ctx, dur, dial)
	},
}

/*NewIDoIO returns a struct the conforms to the IOStreamer interface*/
func NewIDoIO(ctx context.Context, timeout time.Duration, dial string) (IDoIO, error) {
	for re, funcptr := range known {
		if re.MatchString(dial) {
			return funcptr(ctx, timeout, dial)
		}
	}
	err := newErr(false, false, fmt.Errorf("No known way to create a IOStreamer from %q", dial))
	return InvalidIO(err.Error()), err
}
