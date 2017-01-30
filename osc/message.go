package osc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
)

// Message represents a single OSC message. An OSC message consists of an OSC
// address pattern and zero or more arguments.
type Message struct {
	Address   string
	Arguments []interface{}
	addr      net.Addr // Source address of packet.
}

// Verify that interfaces are implemented properly.
var _ Packet = (*Message)(nil)

// NewMessage returns a new Message. The `addr` parameter is the OSC address.
func NewMessage(addr string, args ...interface{}) *Message {
	return &Message{Address: addr, Arguments: args}
}

// Addr implements the Packet interface.
func (msg *Message) Addr() net.Addr { return msg.addr }

// SetAddr implements the Packet interface.
func (msg *Message) SetAddr(addr net.Addr) { msg.addr = addr }

// Append appends the given arguments to the arguments list.
func (msg *Message) Append(args ...interface{}) {
	msg.Arguments = append(msg.Arguments, args...)
}

// Equals returns true if the given OSC Message `m` is equal to the current OSC
// Message. It checks if the OSC address and the arguments are equal. Returns
// true if the current object and `m` are equal.
func (msg *Message) Equals(m *Message) bool {
	return reflect.DeepEqual(msg, m)
}

// Clear clears the OSC address and all arguments.
func (msg *Message) Clear() {
	msg.Address = ""
	msg.ClearData()
}

// ClearData removes all arguments from the OSC Message.
func (msg *Message) ClearData() {
	msg.Arguments = msg.Arguments[len(msg.Arguments):]
}

// Match returns true, if the address of the OSC Message matches the given
// address. The match is case sensitive!
func (msg *Message) Match(addr string) bool {
	exp := getRegEx(msg.Address)
	if exp.MatchString(addr) {
		return true
	}
	return false
}

// TypeTags returns the type tag string.
func (msg *Message) TypeTags() (string, error) {
	if msg == nil {
		return "", fmt.Errorf("message is nil")
	}

	tags := ","
	for _, m := range msg.Arguments {
		s, err := getTypeTag(m)
		if err != nil {
			return "", err
		}
		tags += s
	}

	return tags, nil
}

// String implements the fmt.Stringer interface.
func (msg *Message) String() string {
	if msg == nil {
		return ""
	}

	tags, err := msg.TypeTags()
	if err != nil {
		return ""
	}

	formatString := "%s %s"
	var args []interface{}
	args = append(args, msg.Address)
	args = append(args, tags)

	for _, arg := range msg.Arguments {
		switch arg.(type) {
		case bool, int32, int64, float32, float64, string:
			formatString += " %v"
			args = append(args, arg)

		case nil:
			formatString += " %s"
			args = append(args, "Nil")

		case []byte:
			formatString += " %s"
			args = append(args, "blob")

		case Timetag:
			formatString += " %d"
			timeTag := arg.(Timetag)
			args = append(args, timeTag.TimeTag())
		}
	}

	return fmt.Sprintf(formatString, args...)
}

// CountArguments returns the number of arguments.
func (msg *Message) CountArguments() int {
	return len(msg.Arguments)
}

// MarshalBinary serializes the OSC message to a byte buffer. The byte buffer
// has the following format:
// 1. OSC Address Pattern
// 2. OSC Type Tag String
// 3. OSC Arguments
func (msg *Message) MarshalBinary() ([]byte, error) {
	// We can start with the OSC address and add it to the buffer
	data := new(bytes.Buffer)
	if _, err := writePaddedString(msg.Address, data); err != nil {
		return nil, err
	}

	// Type tag string starts with ","
	typetags := []byte{','}

	// Process the type tags and collect all arguments
	payload := new(bytes.Buffer)
	for _, arg := range msg.Arguments {
		// FIXME: Use t instead of arg
		switch t := arg.(type) {
		default:
			return nil, fmt.Errorf("OSC - unsupported type: %T", t)

		case bool:
			if arg.(bool) == true {
				typetags = append(typetags, 'T')
			} else {
				typetags = append(typetags, 'F')
			}

		case nil:
			typetags = append(typetags, 'N')

		case int32:
			typetags = append(typetags, 'i')
			if err := binary.Write(payload, binary.BigEndian, int32(t)); err != nil {
				return nil, err
			}

		case float32:
			typetags = append(typetags, 'f')
			if err := binary.Write(payload, binary.BigEndian, float32(t)); err != nil {
				return nil, err
			}

		case string:
			typetags = append(typetags, 's')
			if _, err := writePaddedString(t, payload); err != nil {
				return nil, err
			}

		case []byte:
			typetags = append(typetags, 'b')
			if _, err := writeBlob(t, payload); err != nil {
				return nil, err
			}

		case int64:
			typetags = append(typetags, 'h')
			if err := binary.Write(payload, binary.BigEndian, int64(t)); err != nil {
				return nil, err
			}

		case float64:
			typetags = append(typetags, 'd')
			if err := binary.Write(payload, binary.BigEndian, float64(t)); err != nil {
				return nil, err
			}

		case Timetag:
			typetags = append(typetags, 't')
			timeTag := arg.(Timetag)
			if _, err := payload.Write(timeTag.ToByteArray()); err != nil {
				return nil, err
			}
		}
	}

	// Write the type tag string to the data buffer
	if _, err := writePaddedString(string(typetags), data); err != nil {
		return nil, err
	}

	// Write the payload (OSC arguments) to the data buffer
	if _, err := data.Write(payload.Bytes()); err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

// getTypeTag returns the OSC type tag for the given argument.
func getTypeTag(arg interface{}) (string, error) {
	switch t := arg.(type) {
	case bool:
		if arg.(bool) {
			return "T", nil
		}
		return "F", nil
	case nil:
		return "N", nil
	case int32:
		return "i", nil
	case float32:
		return "f", nil
	case string:
		return "s", nil
	case []byte:
		return "b", nil
	case int64:
		return "h", nil
	case float64:
		return "d", nil
	case Timetag:
		return "t", nil
	default:
		return "", fmt.Errorf("Unsupported type: %T", t)
	}
}
