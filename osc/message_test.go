package osc

import "testing"

func TestAppend(t *testing.T) {
	oscAddress := "/address"
	message := NewMessage(oscAddress)
	if message.Address != oscAddress {
		t.Errorf("OSC address should be \"%s\" and is \"%s\"", oscAddress, message.Address)
	}

	message.Append("string argument")
	message.Append(123456789)
	message.Append(true)

	if message.CountArguments() != 3 {
		t.Errorf("Number of arguments should be %d and is %d", 3, message.CountArguments())
	}
}

func TestEquals(t *testing.T) {
	msg1 := NewMessage("/address")
	msg2 := NewMessage("/address")
	msg1.Append(1234)
	msg2.Append(1234)
	msg1.Append("test string")
	msg2.Append("test string")

	if !msg1.Equals(msg2) {
		t.Error("Messages should be equal")
	}
}

func TestTypeTags(t *testing.T) {
	for _, tt := range []struct {
		desc string
		msg  *Message
		tags string
		ok   bool
	}{
		{"addr_only", NewMessage("/"), ",", true},
		{"nil", NewMessage("/", nil), ",N", true},
		{"bool_true", NewMessage("/", true), ",T", true},
		{"bool_false", NewMessage("/", false), ",F", true},
		{"int32", NewMessage("/", int32(1)), ",i", true},
		{"int64", NewMessage("/", int64(2)), ",h", true},
		{"float32", NewMessage("/", float32(3.0)), ",f", true},
		{"float64", NewMessage("/", float64(4.0)), ",d", true},
		{"string", NewMessage("/", "5"), ",s", true},
		{"[]byte", NewMessage("/", []byte{'6'}), ",b", true},
		{"two_args", NewMessage("/", "123", int32(456)), ",si", true},
		{"invalid_msg", nil, "", false},
		{"invalid_arg", NewMessage("/foo/bar", 789), "", false},
	} {
		tags, err := tt.msg.TypeTags()
		if err != nil && tt.ok {
			t.Errorf("%s: TypeTags() unexpected error: %s", tt.desc, err)
			continue
		}
		if err == nil && !tt.ok {
			t.Errorf("%s: TypeTags() expected an error")
			continue
		}
		if !tt.ok {
			continue
		}
		if got, want := tags, tt.tags; got != want {
			t.Errorf("%s: TypeTags() = '%s', want = '%s'", tt.desc, got, want)
		}
	}
}

func TestString(t *testing.T) {
	for _, tt := range []struct {
		desc string
		msg  *Message
		str  string
	}{
		{"nil", nil, ""},
		{"addr_only", NewMessage("/foo/bar"), "/foo/bar ,"},
		{"one_addr", NewMessage("/foo/bar", "123"), "/foo/bar ,s 123"},
		{"two_args", NewMessage("/foo/bar", "123", int32(456)), "/foo/bar ,si 123 456"},
	} {
		if got, want := tt.msg.String(), tt.str; got != want {
			t.Errorf("%s: String() = '%s', want = '%s'", tt.desc, got, want)
		}
	}
}

func TestTypeTagsString(t *testing.T) {
	msg := NewMessage("/some/address")
	msg.Append(int32(100))
	msg.Append(true)
	msg.Append(false)

	typeTags, err := msg.TypeTags()
	if err != nil {
		t.Error(err.Error())
	}

	if typeTags != ",iTF" {
		t.Errorf("Type tag string should be ',iTF' and is: %s", typeTags)
	}
}
