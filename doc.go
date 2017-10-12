/*Package agnoio returns one of a few structures that functionall agnostic to
what the actual IO transport is.  These was conceived of in order to abstract out
many hardware dependant stream IO oriented hardware protocols in an agnostic manner.
Think: how can you treat a network socket, serial port, I2C, SPI, and any
sort of bizzare busses all the same?  Things all these protocols have in common
is that they read & write bytes, can be closed, occasionally screw up and need
to be Re-opened.

Purpose


Have you every wanted to do something bizzare as open a serial port, socket, zmq,
UDP datastream, i2c or spi, or named pipe as agnostic as possible?  If so, then
this package is for you.  This package attempts to be a thin vanier over several
different io packages so callers can be mostly agnostic to what the underling
mechanics are.

Implemented


Currently, the following URI schemas are implemented:
  tcp://<host:port> - Outgoing Sockets of type tcp (either v4 or v6)
  tcp4://<host:port> - Outgoing Sockets of type tcp v4
  tcp6://<host:port> - Outgoing Sockets of type tcp v6
  udp://<host:port> - Outgoing Sockets of type udp (either v4 or v6)
  udp4://<host:port> - Outgoing Sockets of type udp v4
  udp6://<host:port> - Outgoing Sockets of type udp v6
  serial://<device>:<baud> - Serial connection
  rs232://<device>:<baud> - Serial connection


Error Handling


It is not prefered that neither IDoIO nor a Arbiter will try to maintain a constan
connection, but rather that when the connection dies / is killed / fails these
errors are passes to the caller who should have a better idea of what to do

*/
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
