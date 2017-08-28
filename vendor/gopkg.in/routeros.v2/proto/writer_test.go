package proto

import (
	"bytes"
	"testing"
)

func TestEncodeLength(t *testing.T) {
	for _, d := range []struct {
		length   int
		rawBytes []byte
	}{
		{0x00000001, []byte{0x01}},
		{0x00000087, []byte{0x80, 0x87}},
		{0x00004321, []byte{0xC0, 0x43, 0x21}},
		{0x002acdef, []byte{0xE0, 0x2a, 0xcd, 0xef}},
		{0x10000080, []byte{0xF0, 0x10, 0x00, 0x00, 0x80}},
	} {
		b := encodeLength(d.length)
		if bytes.Compare(b, d.rawBytes) != 0 {
			t.Fatalf("Expected output %#v for len=%d, got %#v", d.rawBytes, d.length, b)
		}
	}
}
