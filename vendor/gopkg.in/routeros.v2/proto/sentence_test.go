package proto

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestReadWrite(t *testing.T) {
	for i, test := range []struct {
		in  []string
		out string
		tag string
	}{
		{[]string{"!done"}, `[]`, ""},
		{[]string{"!done", ".tag=abc123"}, `[]`, "abc123"},
		{strings.Split("!re =tx-byte=123456789 =only-key", " "), "[{`tx-byte` `123456789`} {`only-key` ``}]", ""},
	} {
		buf := &bytes.Buffer{}
		// Write sentence into buf.
		w := NewWriter(buf)
		for _, word := range test.in {
			w.WriteWord(word)
		}
		w.WriteWord("")
		// Read sentence from buf.
		r := NewReader(buf)
		sen, err := r.ReadSentence()
		if err != nil {
			t.Errorf("#%d: Input(%#q)=%#v", i, test.in, err)
			continue
		}
		x := fmt.Sprintf("%#q", sen.List)
		if x != test.out {
			t.Errorf("#%d: Input(%#q)=%s; want %s", i, test.in, x, test.out)
		}
		if sen.Tag != test.tag {
			t.Errorf("#%d: Input(%#q)=%s; want %s", i, test.in, sen.Tag, test.tag)
		}
	}
}
