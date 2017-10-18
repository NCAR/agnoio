/*Package agnoio provides and interface and some implementers that allow easier
use of low level IO in a way that is mostly agnostic to the actual IO transport.
This was conceived of in order to abstract out many hardware dependant stream IO
oriented hardware protocols in an agnostic manner.  This package attemps to answer
the question of how to treat a network socket, serial port, I2C, SPI, and any other
common streaming systems the same. All these transports have the following in
common: they read & write bytes, can be closed, occasionally screw up and need
to be re-opened (and that is what the IDoIO interface tries to provide).

Purpose

Have you every wanted to do something bizzare as open a serial port, socket, zmq,
UDP datastream, i2c or spi, or named pipe as agnostic as possible?  If so, then
this package is for you.  This package attempts to be a thin vanier over several
different io packages so callers can be mostly agnostic to what the underling
mechanics are.

Interfaces

This package provides two different, but related, interfaces:  IDoIO (eye-do-eye-oh)
and an Arbiter. An IDoIO is the basic read-write-open-close item that can read
and write some bytes off of. Examples of an IDoIO include a serial port connected
to a GPS NMEA (stream of location information) or some a free running sampling
instrument (temperature, distance, speed, height, motor speed, etc) running over
a tcp socket. An Arbiter is similiar in concept to an IDoIO, but it provides more
of a command & control interface:  You send it commands, the remote side does
something with the command, and provides a response. The Arbiter pattern might
be found in sending commands to a servo to move to a different position,
move the center frequency of a piece of test equiptment, or send some sort of
power on/off sequence to a PDU in a data center.  IDoIOs usually need some sort
of parser where Arbiters need to be instructed what to do.


Dial Strings and Implementations


Although you can write your own IDoIO (and I welcome patchs!), the this package
provides IDoIOs for the following transports, which are selected via a URI dial
string.  The schema (eg tcp:// or serial://) portion determines the backend for
which IDoIO implementation is selected. Hereafter, the individual format for
the remaing portion of the string is implementation specific, but should be
transparent enough that someone with a crude undertanding would know what to
make of the parameters.  The following schemas are provided by this package,
and can be generically returned via the NewIDoIO() function:

  tcp://<host:port> - Outgoing Sockets of type tcp (either v4 or v6)
  tcp4://<host:port> - Outgoing Sockets of type tcp v4
  tcp6://<host:port> - Outgoing Sockets of type tcp v6
  udp://<host:port> - Outgoing Sockets of type udp (either v4 or v6)
  udp4://<host:port> - Outgoing Sockets of type udp v4
  udp6://<host:port> - Outgoing Sockets of type udp v6
  serial://<device>:<baud> - Serial connection
  rs232://<device>:<baud> - Serial connection


Context Usage


This package makes use of the context package.  The passed context is used to
derive child contexts and a cancel function.  If .Stop() is called, the cancel
function will be called, and any further IO using the struture will end up in
context errors.  This is helpful as it forces connection hangup and known exit
behaviour.

Error Handling


It is prefered that no structres provided by this package attempt to maintain a
constant connection, but rather that when the connection dies / is killed /
fails / returns errors, the caller should have a bit of knowledge as to what to
do with these errors, such as reconnect, panic, stick a finger in a light socket,
etc.  Generally each transport will have some sort of unique errors that might need
special handling.
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
