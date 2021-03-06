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

import "net"

var _ error = &neterror{}
var _ net.Error = &neterror{}

type neterror struct {
	err                error
	temporary, timeout bool
}

//newErr returns an error that conforms to net.Error
func newErr(temporary, timeout bool, err error) *neterror {
	return &neterror{
		err:       err,
		temporary: temporary,
		timeout:   timeout,
	}
}

/*Error returns the base error as a string, and conforms to the error interface */
func (ne neterror) Error() string {
	return ne.err.Error()
}

/*Temporary return true if the error is a temporary error, indicating the connection
is still active, and the error */
func (ne neterror) Temporary() bool {
	return ne.temporary
}

func (ne neterror) Timeout() bool {
	return ne.timeout
}

/*IsTemporary is a shorthand way to check if a returned error is temporary. Dont
pass nil errors here, the desired behaviour is not defined, and will panic*/
func IsTemporary(err error) bool {
	if err == nil {
		panic("Unable to determine what to do with a nil error.")
	}
	if ne, ok := err.(net.Error); ok {
		return ne.Temporary()
	}
	return false
}

/*IsTimeout is a shorthand way to check if a returned error is a timeout. Dont
pass nil errors here, the desired behaviour is not defined, and will panic*/
func IsTimeout(err error) bool {
	if err == nil {
		panic("Unable to determine what to do with a nil error.")
	}
	if ne, ok := err.(net.Error); ok {
		return ne.Timeout()
	}
	return false
}
