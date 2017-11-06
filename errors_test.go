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
	"errors"
	"testing"
)

func TestNetError(t *testing.T) {
	e := newErr(true, true, errors.New("wwoohoo"))
	_ = e.Error()
	_ = e.Timeout()
	_ = e.Temporary()
	if !IsTimeout(e) || !IsTemporary(e) {
		t.Error("Expected e to be a timeout and temporary")
	}

	ee := errors.New("Boring error")
	if IsTimeout(ee) || IsTemporary(ee) {
		t.Error("Expected e to be neither a timeout nor temporary")
	}

	//catch panics
	f := func(p func(error) bool) {
		var e interface{}
		defer func() {
			e = recover()
			if e == nil {
				t.Error("expected a panic on sending a nil error")
			}
		}()
		p(nil)
	}

	f(IsTimeout)
	f(IsTemporary)
}
