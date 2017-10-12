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
	"fmt"
	"regexp"
	"strings"
	"time"
)

/*Command represents a command represents the Command portoin of a Command-Response operation.
 */
type Command struct {
	/*Name is the human name of command, typically without any arugments. EG if
	the Prototype is something like "Floor\x55\x32", the name should be something
	that make sense for your average human being:  like "Go to Floor 55"*/
	Name string

	/*Timeout is the max time allowed before the command should be forced to return
	return a failed-because-it-took-too-long response. If the command take longer
	than this timeout, the command is to be understood to have failed*/
	Timeout time.Duration

	/*Prototype is the command prototype that is fed, with any arguments, to fmt.Sprintf
	and converted to bytes to shovel to a IDoIO.  That is,
	    fmt(.Prototype, args...)
	is sent down the line.*/
	Prototype string

	/*CommandRegexp is the regex that the final command must match before being
	returned by byes. This works in conjunction with the .Prototype in the
	following way such that c, defined by the following:
	     c := fmt.Sprintf(.Prototype, v ... interface{})
	must not contain %!, (a sign of too many/few/wrong parameters), and
	     CommandRegexp.MatchString(c)
	must be true.*/
	CommandRegexp *regexp.Regexp

	//Response is a regexp that should match good/positive/affirmative responses.
	Response *regexp.Regexp

	//Error is a regexp that should match bad/negative/failure responses
	Error *regexp.Regexp

	//Description is a human readable string of a brief explaination of the commands purpose
	Description string
}

/*Response is what is returns from Command requests.

Bytes is a copy of the []byte read while waiting for a timeout or matching response.
Error is one of:
  - nil if the bytes received match the ingoing Command.Response regexp
  - ErrTimeout if a timeout was receieved
  - Some other error on other low level issures.
Duration is the duration the command took before it succeeded (or failed).
*/
type Response struct {
	Bytes    []byte        //Raw bytes read or received.  In Control funcs, this is the raw value that matched the 'match' clause
	Error    error         //any non-nil errors
	Duration time.Duration //how long did the request take
}

//String implements the Stringer interface
func (r Response) String() string {
	return fmt.Sprintf("Response> Rx Bytes: %q\tErrors: %v\tDuration: %v", r.Bytes, r.Error, r.Duration)
}

//String implements the Stringer interface
func (c Command) String() string {
	sanitize := func(i interface{}) string {
		var str string
		switch s := i.(type) {
		case *regexp.Regexp:
			if s == nil {
				return "nil"
			}
			str = s.String()
		case string:
			str = s
		}
		return strings.Replace(strings.Replace(str, "\r", "\\r", -1), "\n", "\\n", -1)
	}
	return fmt.Sprintf("%s: %v Prototype:%q CommandRegexp:%q Expect:%q Error:%q", c.Name, c.Timeout, sanitize(c.Prototype), sanitize(c.CommandRegexp), sanitize(c.Response), sanitize(c.Error))
}

//ErrBytesArgs is returned when calling Bytes if any of the following occur:
//	Wrong Number of args (too few / many)
//	Wrong order (ie Command.Prototype is "%s %d" and provided args are '24, "string"'')
//	Wrong types (ie Command.Prototype is "%s" and provided arg is '25')
var ErrBytesArgs = fmt.Errorf("Proper arguments not provided to expand command into bytes")

//ErrBytesFormat is returned when the args used to populate the command are forming an invalid command
var ErrBytesFormat = fmt.Errorf("Formed command does not match allowable format for outgoing commands")

/*Bytes returnes the raw bytes that should be sent to the interface based on the Command.Prototype and
any optional arguments passed to it. It will return a byte slice and one of the following errors:

	ErrBytesArgs if either too many, not enough, or the wrong type of args are provided
	ErrBytesFormat if the assembled byte slice does not match the required Command.CommandRegexp
	nil if a byte slice was successfully formed
*/
func (c Command) Bytes(v ...interface{}) ([]byte, error) {
	str := fmt.Sprintf(c.Prototype, v...)
	if strings.Contains(str, "%!") {
		return []byte(str), ErrBytesArgs
	}
	//make sure whatever we stuffed matches the provided regexp
	if !c.CommandRegexp.MatchString(str) {
		return []byte(str), ErrBytesFormat
	}
	return []byte(str), nil

}

//Commands is map of Command structure where the key should be Command.Name
type Commands map[string]Command

//String implements the Stringer() interface
func (c Commands) String() (r string) {
	for _, val := range c {
		r += fmt.Sprintf("%s\n", val.String())
	}
	return
}

//JSONLabels returns a json array of the stored commands
func (c Commands) JSONLabels() (r string) {
	r = "["
	i := 0
	for lab := range c {
		switch i {
		default:
			r += ","
		case 0:
		}
		i++
		r += fmt.Sprintf("%q", lab)
	}
	r += "]"
	return
}

/*Merge takes multiple command sets and returns a single command set*/
func Merge(cmds ...Commands) Commands {
	c := Commands{}
	for _, cmdset := range cmds {
		for name, cmd := range cmdset {
			c[name] = cmd
		}
	}
	return c
}
