package osc

import (
	"bufio"
	"bytes"
	"encoding"
	"net"
)

// Packet is the interface for Message and Bundle.
type Packet interface {
	encoding.BinaryMarshaler

	// Addr returns the source address of the packet.
	Addr() net.Addr
	// SetAddr sets the source address of the packet.
	SetAddr(net.Addr)
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
