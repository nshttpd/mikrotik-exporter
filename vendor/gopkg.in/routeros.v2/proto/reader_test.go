package proto

import (
	"bytes"
	"testing"
)

func TestReadLength(t *testing.T) {
	for _, d := range []struct {
		length   int64
		rawBytes []byte
	}{
		{0x00000001, []byte{0x01}},
		{0x00000087, []byte{0x80, 0x87}},
		{0x00004321, []byte{0xC0, 0x43, 0x21}},
		{0x002acdef, []byte{0xE0, 0x2a, 0xcd, 0xef}},
		{0x10000080, []byte{0xF0, 0x10, 0x00, 0x00, 0x80}},
	} {
		r := NewReader(bytes.NewBuffer(d.rawBytes)).(*reader)
		l, err := r.readLength()
		if err != nil {
			t.Fatalf("readLength error: %s", err)
		}
		if l != d.length {
			t.Fatalf("Expected len=%X for input %#v, got %X", d.length, d.rawBytes, l)
		}
	}
}
