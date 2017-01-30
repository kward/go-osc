// go-osc provides a package for sending and receiving OpenSoundControl
// messages. The package is implemented in pure Go.
package osc

import (
	"bufio"
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Packet is the interface for Message and Bundle.
type Packet interface {
	encoding.BinaryMarshaler

	// Addr returns the source address of the packet.
	Addr() net.Addr
	// SetAddr sets the source address of the packet.
	SetAddr(net.Addr)
}

// Handler is an interface for message handlers. Every handler implementation
// for an OSC message must implement this interface.
type Handler interface {
	HandleMessage(msg *Message)
}

// HandlerFunc implements the Handler interface. Type definition for an OSC
// handler function.
type HandlerFunc func(msg *Message)

// HandleMessage calls itself with the given OSC Message. Implements the
// Handler interface.
func (f HandlerFunc) HandleMessage(msg *Message) {
	f(msg)
}

// ParsePacket reads the packet from a message.
func ParsePacket(msg string) (Packet, error) {
	var start int
	return readPacket(bufio.NewReader(bytes.NewBufferString(msg)), &start, len(msg))
}

// receivePacket receives an OSC packet from the given reader.
func readPacket(reader *bufio.Reader, start *int, end int) (Packet, error) {
	buf, err := reader.Peek(1)
	if err != nil {
		return nil, err
	}

	// An OSC Message starts with a '/'
	if buf[0] == '/' {
		pkt, err := readMessage(reader, start)
		if err != nil {
			return nil, err
		}
		return pkt, err
	}
	if buf[0] == '#' { // An OSC bundle starts with a '#'
		pkt, err := readBundle(reader, start, end)
		if err != nil {
			return nil, err
		}
		return pkt, nil
	}

	var pkt Packet
	return pkt, nil
}

// readBundle reads an Bundle from reader.
func readBundle(reader *bufio.Reader, start *int, end int) (*Bundle, error) {
	// Read the '#bundle' OSC string
	startTag, n, err := readPaddedString(reader)
	if err != nil {
		return nil, err
	}
	*start += n

	if startTag != bundleTag {
		return nil, fmt.Errorf("Invalid bundle start tag: %s", startTag)
	}

	// Read the timetag
	var timeTag uint64
	if err := binary.Read(reader, binary.BigEndian, &timeTag); err != nil {
		return nil, err
	}
	*start += 8

	// Create a new bundle
	bundle := NewBundle(timetagToTime(timeTag))

	// Read until the end of the buffer
	for *start < end {
		// Read the size of the bundle element
		var length int32
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			return nil, err
		}
		*start += 4

		pkt, err := readPacket(reader, start, end)
		if err != nil {
			return nil, err
		}
		if err = bundle.Append(pkt); err != nil {
			return nil, err
		}
	}

	return bundle, nil
}

// readMessage from `reader`.
func readMessage(reader *bufio.Reader, start *int) (*Message, error) {
	// First, read the OSC address
	addr, n, err := readPaddedString(reader)
	if err != nil {
		return nil, err
	}
	*start += n

	// Read all arguments
	msg := NewMessage(addr)
	if err = readArguments(msg, reader, start); err != nil {
		return nil, err
	}

	return msg, nil
}

// readArguments from `reader` and add them to the OSC message `msg`.
func readArguments(msg *Message, reader *bufio.Reader, start *int) error {
	// Read the type tag string
	var n int
	typetags, n, err := readPaddedString(reader)
	if err != nil {
		return err
	}
	*start += n

	// If the typetag doesn't start with ',', it's not valid
	if typetags[0] != ',' {
		return errors.New("unsupported type tag string")
	}

	// Remove ',' from the type tag
	typetags = typetags[1:]

	for _, c := range typetags {
		switch c {
		default:
			return fmt.Errorf("unsupported type tag: %c", c)

		case 'i': // int32
			var i int32
			if err = binary.Read(reader, binary.BigEndian, &i); err != nil {
				return err
			}
			*start += 4
			msg.Append(i)

		case 'h': // int64
			var i int64
			if err = binary.Read(reader, binary.BigEndian, &i); err != nil {
				return err
			}
			*start += 8
			msg.Append(i)

		case 'f': // float32
			var f float32
			if err = binary.Read(reader, binary.BigEndian, &f); err != nil {
				return err
			}
			*start += 4
			msg.Append(f)

		case 'd': // float64/double
			var d float64
			if err = binary.Read(reader, binary.BigEndian, &d); err != nil {
				return err
			}
			*start += 8
			msg.Append(d)

		case 's': // string
			// TODO: fix reading string value
			var s string
			if s, _, err = readPaddedString(reader); err != nil {
				return err
			}
			*start += len(s) + padBytesNeeded(len(s))
			msg.Append(s)

		case 'b': // blob
			var buf []byte
			var n int
			if buf, n, err = readBlob(reader); err != nil {
				return err
			}
			*start += n
			msg.Append(buf)

		case 't': // OSC time tag
			var tt uint64
			if err = binary.Read(reader, binary.BigEndian, &tt); err != nil {
				return nil
			}
			*start += 8
			msg.Append(NewTimetagFromTimetag(tt))

		case 'T': // true
			msg.Append(true)

		case 'F': // false
			msg.Append(false)
		}
	}

	return nil
}

////
// De/Encoding functions
////

// readBlob reads an OSC blob from the blob byte array. Padding bytes are
// removed from the reader and not returned.
func readBlob(reader *bufio.Reader) ([]byte, int, error) {
	// First, get the length
	var blobLen int
	if err := binary.Read(reader, binary.BigEndian, &blobLen); err != nil {
		return nil, 0, err
	}
	n := 4 + blobLen

	// Read the data
	blob := make([]byte, blobLen)
	if _, err := reader.Read(blob); err != nil {
		return nil, 0, err
	}

	// Remove the padding bytes
	numPadBytes := padBytesNeeded(blobLen)
	if numPadBytes > 0 {
		n += numPadBytes
		dummy := make([]byte, numPadBytes)
		if _, err := reader.Read(dummy); err != nil {
			return nil, 0, err
		}
	}

	return blob, n, nil
}

// writeBlob writes the data byte array as an OSC blob into buff. If the length
// of data isn't 32-bit aligned, padding bytes will be added.
func writeBlob(data []byte, buf *bytes.Buffer) (int, error) {
	// Add the size of the blob
	dlen := int32(len(data))
	if err := binary.Write(buf, binary.BigEndian, dlen); err != nil {
		return 0, err
	}

	// Write the data
	if _, err := buf.Write(data); err != nil {
		return 0, nil
	}

	// Add padding bytes if necessary
	numPadBytes := padBytesNeeded(len(data))
	if numPadBytes > 0 {
		padBytes := make([]byte, numPadBytes)
		n, err := buf.Write(padBytes)
		if err != nil {
			return 0, err
		}
		numPadBytes = n
	}

	return 4 + len(data) + numPadBytes, nil
}

// readPaddedString reads a padded string from the given reader. The padding
// bytes are removed from the reader.
func readPaddedString(reader *bufio.Reader) (string, int, error) {
	// Read the string from the reader
	str, err := reader.ReadString(0)
	if err != nil {
		return "", 0, err
	}
	n := len(str)

	// Remove the string delimiter, in order to calculate the right amount
	// of padding bytes
	str = str[:len(str)-1]

	// Remove the padding bytes
	padLen := padBytesNeeded(len(str)) - 1
	if padLen > 0 {
		n += padLen
		padBytes := make([]byte, padLen)
		if _, err = reader.Read(padBytes); err != nil {
			return "", 0, err
		}
	}

	return str, n, nil
}

// writePaddedString writes a string with padding bytes to the a buffer.
// Returns, the number of written bytes and an error if any.
func writePaddedString(str string, buf *bytes.Buffer) (int, error) {
	// Write the string to the buffer
	n, err := buf.WriteString(str)
	if err != nil {
		return 0, err
	}

	// Calculate the padding bytes needed and create a buffer for the padding bytes
	numPadBytes := padBytesNeeded(len(str))
	if numPadBytes > 0 {
		padBytes := make([]byte, numPadBytes)
		// Add the padding bytes to the buffer
		n, err := buf.Write(padBytes)
		if err != nil {
			return 0, err
		}
		numPadBytes = n
	}

	return n + numPadBytes, nil
}

// padBytesNeeded determines how many bytes are needed to fill up to the next 4
// byte length.
func padBytesNeeded(elementLen int) int {
	return 4*(elementLen/4+1) - elementLen
}

////
// Utility and helper functions
////

// getRegEx compiles and returns a regular expression object for the given
// address `pattern`.
func getRegEx(pattern string) *regexp.Regexp {
	for _, trs := range []struct {
		old, new string
	}{
		{".", `\.`}, // Escape all '.' in the pattern
		{"(", `\(`}, // Escape all '(' in the pattern
		{")", `\)`}, // Escape all ')' in the pattern
		{"*", ".*"}, // Replace a '*' with '.*' that matches zero or more chars
		{"{", "("},  // Change a '{' to '('
		{",", "|"},  // Change a ',' to '|'
		{"}", ")"},  // Change a '}' to ')'
		{"?", "."},  // Change a '?' to '.'
	} {
		pattern = strings.Replace(pattern, trs.old, trs.new, -1)
	}

	return regexp.MustCompile(pattern)
}
