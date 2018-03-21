package routeros

import (
	"flag"
	"strings"
	"testing"
	"time"
)

var (
	routerosAddress  = flag.String("routeros.address", "", "RouterOS address:port")
	routerosUsername = flag.String("routeros.username", "admin", "RouterOS user name")
	routerosPassword = flag.String("routeros.password", "admin", "RouterOS password")
)

type liveTest struct {
	*testing.T
	c *Client
}

func newLiveTest(t *testing.T) *liveTest {
	tt := &liveTest{T: t}
	tt.connect()
	return tt
}

func (t *liveTest) connect() {
	if *routerosAddress == "" {
		t.Skip("Flag -routeros.address not set")
	}
	var err error
	t.c, err = Dial(*routerosAddress, *routerosUsername, *routerosPassword)
	if err != nil {
		t.Fatal(err)
	}
}

func (t *liveTest) run(sentence ...string) *Reply {
	t.Logf("Run: %#q", sentence)
	r, err := t.c.RunArgs(sentence)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Reply: %s", r)
	return r
}

func (t *liveTest) getUptime() {
	r := t.run("/system/resource/print")
	if len(r.Re) != 1 {
		t.Fatalf("len(!re)=%d; want 1", len(r.Re))
	}
	_, ok := r.Re[0].Map["uptime"]
	if !ok {
		t.Fatal("Missing uptime")
	}
}

func TestRunSync(tt *testing.T) {
	t := newLiveTest(tt)
	defer t.c.Close()
	t.getUptime()
}

func TestRunAsync(tt *testing.T) {
	t := newLiveTest(tt)
	defer t.c.Close()
	t.c.Async()
	t.getUptime()
}

func TestRunError(tt *testing.T) {
	t := newLiveTest(tt)
	defer t.c.Close()
	for i, sentence := range [][]string{
		{"/xxx"},
		{"/ip/address/add", "=address=127.0.0.2/32", "=interface=xxx"},
	} {
		t.Logf("#%d: Run: %#q", i, sentence)
		_, err := t.c.RunArgs(sentence)
		if err == nil {
			t.Error("Success; want error from RouterOS device trying to run an invalid command")
		}
	}
}

func TestDialInvalidPort(t *testing.T) {
	c, err := Dial("127.0.0.1:xxx", "x", "x")
	if err == nil {
		c.Close()
		t.Fatalf("Dial succeeded; want error")
	}
	if err.Error() != "dial tcp: lookup tcp/xxx: getaddrinfow: The specified class was not found." {
		t.Fatal(err)
	}
}

func TestDialTLSInvalidPort(t *testing.T) {
	c, err := DialTLS("127.0.0.1:xxx", "x", "x", nil)
	if err == nil {
		c.Close()
		t.Fatalf("Dial succeeded; want error")
	}
	if err.Error() != "dial tcp: lookup tcp/xxx: getaddrinfow: The specified class was not found." {
		t.Fatal(err)
	}
}

func TestDialTimeout(t *testing.T) {
	c, err := DialTimeout("127.0.0.2:8729", "x", "x", time.Second)
	if err == nil {
		c.Close()
		t.Fatalf("DialTimeout succeeded; want error")
	}

	if !strings.Contains(err.Error(), "i/o timeout") {
		t.Fatalf("DialTimeout: Timeout expected in err. Has: %s", err)
	}
}

func TestDialTLSTimeout(t *testing.T) {
	c, err := DialTLSTimeout("127.0.0.2:8729", "x", "x", nil, time.Second)
	if err == nil {
		c.Close()
		t.Fatalf("DialTLSTimeout succeeded; want error")
	}

	if !strings.Contains(err.Error(), "i/o timeout") {
		t.Fatalf("DialTLSTimeout: Timeout expected in err. Has: %s", err)
	}
}

func TestInvalidLogin(t *testing.T) {
	if *routerosAddress == "" {
		t.Skip("Flag -routeros.address not set")
	}
	var err error
	c, err := Dial(*routerosAddress, "xxx", "APasswordThatWillNeverExistir")
	if err == nil {
		c.Close()
		t.Fatalf("Dial succeeded; want error")
	}
	if err.Error() != "from RouterOS device: cannot log in" {
		t.Fatal(err)
	}
}
