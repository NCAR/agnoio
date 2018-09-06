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
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
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

/*sanitize turns derenders ASCII control seq to to readable equivalents*/
func sanitize(i interface{}) string {
	var str string
	switch s := i.(type) {
	case *regexp.Regexp:
		if s == nil {
			return "-"
		}
		str = s.String()
	case string:
		str = s
	}
	return strings.Replace(strings.Replace(str, "\r", "\\r", -1), "\n", "\\n", -1)
}

//String implements the Stringer interface
func (c Command) String() string {
	return fmt.Sprintf("%s: %v Prototype:%q CommandRegexp:%q Expect:%q Error:%q", c.Name, c.Timeout, sanitize(c.Prototype), sanitize(c.CommandRegexp), sanitize(c.Response), sanitize(c.Error))
}

/*Bytes returnes the raw bytes that should be sent to the interface based on the
Command.Prototype and any optional arguments passed to it via
  fmt.Sprintf(.Prototype, v...)
If the resulting string formed by above contains any "%!" sequences, then this
assumes that the formed command was not properly fed through fmt.Sprintf, and will
return the package error ErrBytesArgs. This currently does not allow for embeded "#!"
sequences, which should be fixed via lexical analysis

If .CommandRegexp is nil, it is assumed that any command formed (sans the above rule)
is acceptable.  If not, the formed command is compared against CommandRegexp.  If
the fomed command does not match, the package error ErrBytesFormat is returned.

If all goes well, a byte slice to be sent down the line and a nil error is returned.

BUG: Current implementation disallows handling of commands with "%!" sequences
*/
func (c Command) Bytes(v ...interface{}) ([]byte, error) {
	str := fmt.Sprintf(c.Prototype, v...)
	//checking for wrong, or invalid arguments
	if strings.Contains(str, "%!") {
		return []byte(str), ErrBytesArgs
	}
	//make sure whatever we stuffed matches the provided regexp
	if c.CommandRegexp != nil && !c.CommandRegexp.MatchString(str) {
		return []byte(str), ErrBytesFormat
	}
	return []byte(str), nil

}

//Commands is map of Command structure where the key should be Command.Name
type Commands map[string]Command

//String implements the Stringer() interface
func (c Commands) String() (r string) {
	cmds := sort.StringSlice{}
	for cmd := range c {
		cmds = append(cmds, cmd)
	}
	cmds.Sort()

	buf := bytes.NewBufferString("")
	tw := tablewriter.NewWriter(buf)
	tw.SetAutoWrapText(false)
	tw.SetHeader([]string{"Name", "Timeout", "Prototype", "Command Regex", "Resp Regex", "Error Regex"})

	for _, cc := range cmds {
		cmd := c[cc]
		tw.Append([]string{
			cc,
			cmd.Timeout.String(),
			sanitize(cmd.Prototype),
			sanitize(cmd.CommandRegexp),
			sanitize(cmd.Response),
			sanitize(cmd.Error),
		})
	}
	tw.Render()
	return buf.String()
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

/*Contains returns true if the command set contains any of the passed named
commands.  It checks the key values, not the embedded Command.Name values*/
func (c Commands) Contains(named ...string) bool {
	if c == nil || len(named) == 0 {
		return false
	}
	for _, name := range named {
		if _, ok := c[name]; !ok {
			return false
		}
	}
	return true
}

/*Clone returns a deep copy of the Commands*/
func (c Commands) Clone() Commands {
	r := Commands{}
	for name, cmd := range c {
		r[name] = cmd
	}
	return r
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
