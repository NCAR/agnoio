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
	"encoding/json"
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestCommand_Bytes(t *testing.T) {
	//test Variadic Bytes

	singular := Command{
		Name:          "ping",
		Timeout:       time.Duration(500) * time.Millisecond,
		Prototype:     "\r",
		CommandRegexp: regexp.MustCompile("\r"),
	}
	d, err := singular.Bytes()
	if err != nil {
		t.Fatalf("Command without args should not have an error: %v", err)
	}
	t.Logf("Command #1 rendered to %q", d)

	arged := Command{
		Name:          "ping #2",
		Timeout:       time.Duration(500) * time.Millisecond,
		Prototype:     "%02d\r",
		CommandRegexp: regexp.MustCompile("[0-9]{2}\r"),
	}

	d, err = arged.Bytes()
	if err == nil {
		t.Fatalf("Command with arg didnt return error when it should have.")
	}
	t.Logf("Command #2 rendered to %q (should be erroneous)", d)

	d, err = arged.Bytes(13)
	if err != nil {
		t.Fatalf("Command with arg didnt properly format with proper number of args")
	}
	t.Logf("Command #2 rendered to %q", d)

	d, err = arged.Bytes(13, 5)
	if err == nil {
		t.Fatalf("Command with too many args should error out")
	}
	t.Logf("Command #2 rendered to %q", d)

	badformatted := Command{
		Name:          "ping #2",
		Timeout:       time.Duration(500) * time.Millisecond,
		Prototype:     "%02x\r",
		CommandRegexp: regexp.MustCompile("[0-9]{2}\r"),
	}

	_, err = badformatted.Bytes(255)
	if err == nil {
		t.Fatalf("Mismatched prototype and command regexp matched")
	}

	d, err = badformatted.Bytes(15)
	if err == nil {
		t.Logf("%v", d)
		t.Fatalf("Mismatched prototype and command regexp matched")
	}

}
func TestCommand_String(t *testing.T) {
	cmds := map[string]Command{
		`p: 1s Prototype:"p" CommandRegexp:"" Expect:"" Error:""`: Command{
			Name:          "p",
			Timeout:       1 * time.Second,
			Prototype:     "p",
			CommandRegexp: regexp.MustCompile(""),
			Error:         regexp.MustCompile(""),
			Response:      regexp.MustCompile(""),
		},
		`q: 1s Prototype:"q" CommandRegexp:"" Expect:"nil" Error:""`: Command{
			Name:          "q",
			Timeout:       1 * time.Second,
			Prototype:     "q",
			CommandRegexp: regexp.MustCompile(""),
			Error:         regexp.MustCompile(""),
			Response:      nil,
		},
	}
	for val, cmd := range cmds {
		if val != cmd.String() {
			t.Fatalf("Not formatting '%s' into '%s'", cmd.String(), val)
		}
	}

}

var testCommands = Commands{
	"test": Command{Name: "test"},
	"ping": Command{Name: "ping"},
}

func TestCommands_String(t *testing.T) {
	cmds := Commands{
		"p": Command{
			Name:          "p",
			Timeout:       1 * time.Second,
			Prototype:     "p",
			CommandRegexp: regexp.MustCompile(""),
			Error:         regexp.MustCompile(""),
			Response:      regexp.MustCompile(""),
		},
		"q": Command{
			Name:          "q",
			Timeout:       1 * time.Second,
			Prototype:     "q",
			CommandRegexp: regexp.MustCompile(""),
			Error:         regexp.MustCompile(""),
			Response:      regexp.MustCompile(""),
		},
	}
	is := cmds.String()
	shouldbeA := cmds["p"].String() + "\n" + cmds["q"].String() + "\n"
	shouldbeB := cmds["q"].String() + "\n" + cmds["p"].String() + "\n"
	if is != shouldbeA && is != shouldbeB {
		t.Fatalf("Multiple command formatting didnt render properly:\nGot :`%s`\nWant: `%s`\nOr  :`%s`", is, shouldbeA, shouldbeB)
	}
}

func TestCommands_JSONLabels(t *testing.T) {
	t.Log("Testing Commands_JSONLabels")

	tests := []Commands{
		testCommands,
	}
	for _, cmdset := range tests {
		js := cmdset.JSONLabels()
		var v []string
		if e := json.Unmarshal([]byte(js), &v); e != nil {
			t.Fatalf(`Emitted json "%s" isnt valid: %v`, js, e)
			t.FailNow()
		}
		t.Log("v=", v)
		for key := range cmdset {
			t.Log("Checking for ", key, "in json output")
			notfound := true
			for _, vv := range v {
				if vv == key {
					notfound = false
				}
			}
			if notfound {
				t.Errorf("Unable to located %s in returned JSON list %s", key, js)
			}
		}
	}
}

func TestResponse_String(t *testing.T) {
	var resp Response
	if resp.String() != `Response> Rx Bytes: ""	Errors: <nil>	Duration: 0s` {
		t.Logf("got\n%s\n", resp.String())
		t.Fatalf("Response String() func not working")
	}
	resp.Bytes = []byte("a")
	resp.Duration = 1 * time.Second
	if resp.String() != `Response> Rx Bytes: "a"	Errors: <nil>	Duration: 1s` {
		t.Fatalf("Response String() func not working")
	}
}

func TestCommands_Contains(t *testing.T) {
	c, d := Commands(nil), Commands{}
	if c.Contains() || d.Contains() {
		t.Error("nil & empty Commands should Contain() false")
	}
	c = Commands{"a": Command{}, "b": Command{}}
	if c.Contains("a", "b", "c") {
		t.Error("Expect contains to return true for all values")
	}
	if !c.Contains("a", "b") {
		t.Error("Expected true")
	}
}

func TestCommands_Clone(t *testing.T) {
	c := Commands{"a": Command{}, "b": Command{}}
	d := c.Clone()

	delete(d, "a")
	delete(d, "b")
	if d.Contains("a", "b") || !c.Contains("a", "b") {
		t.Error("Clone should not be coupled to the parent")
	}

	//clean out d

}

func TestMerge(t *testing.T) {
	c := Commands{"a": Command{}, "b": Command{}}
	d := Merge(c, c, c, c, c, c, c)
	if !reflect.DeepEqual(c, d) {
		t.Errorf("Didnt munge properly")
	}
}
