package routeros_test

import (
	"fmt"
	"io"
	"testing"

	"gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"
)

func TestLogin(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/login @ []")
		s.writeSentence(t, "!done", "=ret=abc123")
		s.readSentence(t, "/login @ [{`name` `userTest`} {`response` `0021277bff9ac7caf06aa608e46616d47f`}]")
		s.writeSentence(t, "!done")
	}()

	err := c.Login("userTest", "passTest")
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoginIncorrect(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/login @ []")
		s.writeSentence(t, "!done", "=ret=abc123")
		s.readSentence(t, "/login @ [{`name` `userTest`} {`response` `0021277bff9ac7caf06aa608e46616d47f`}]")
		s.writeSentence(t, "!trap", "=message=incorrect login")
	}()

	err := c.Login("userTest", "passTest")
	if err == nil {
		t.Fatalf("Login succeeded; want error")
	}
	if err.Error() != "from RouterOS device: incorrect login" {
		t.Fatal(err)
	}
}

func TestLoginNoChallenge(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/login @ []")
		s.writeSentence(t, "!done")
	}()

	err := c.Login("userTest", "passTest")
	if err == nil {
		t.Fatalf("Login succeeded; want error")
	}
	if err.Error() != "RouterOS: /login: no ret (challenge) received" {
		t.Fatal(err)
	}
}

func TestLoginInvalidChallenge(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/login @ []")
		s.writeSentence(t, "!done", "=ret=Invalid Hex String")
	}()

	err := c.Login("userTest", "passTest")
	if err == nil {
		t.Fatalf("Login succeeded; want error")
	}
	if err.Error() != "RouterOS: /login: invalid ret (challenge) hex string received: encoding/hex: invalid byte: U+0049 'I'" {
		t.Fatal(err)
	}
}

func TestLoginEOF(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()
	s.Close()

	err := c.Login("userTest", "passTest")
	if err == nil {
		t.Fatalf("Login succeeded; want error")
	}
	if err.Error() != "io: read/write on closed pipe" {
		t.Fatal(err)
	}
}

func TestCloseTwice(t *testing.T) {
	c, s := newPair(t)
	defer s.Close()
	c.Close()
	c.Close()
}

func TestAsyncTwice(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()
	defer s.Close()

	c.Async()

	errC := c.Async()
	err := <-errC
	want := "Async() has already been called"
	if err.Error() != want {
		t.Fatalf("Second Async()=%#q; want %#q", err, want)
	}

	err = <-errC
	if err != nil {
		t.Fatalf("Async() channel should be closed after error; got %#q", err)
	}
}

func TestRun(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!re", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done")
	}()

	sen, err := c.Run("/ip/address")
	if err != nil {
		t.Fatal(err)
	}
	want := "!re @ [{`address` `1.2.3.4/32`}]\n!done @ []"
	if sen.String() != want {
		t.Fatalf("/ip/address (%s); want (%s)", sen, want)
	}
}

func TestRunWithListen(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @l1 []")
		s.writeSentence(t, "!re", ".tag=l1", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done", ".tag=l1")
	}()

	listen, err := c.Listen("/ip/address")
	if err != nil {
		t.Fatal(err)
	}

	sen := <-listen.Chan()
	want := "!re @l1 [{`address` `1.2.3.4/32`}]"
	if fmt.Sprintf("%s", sen) != want {
		t.Fatalf("/ip/address (%s); want (%s)", sen, want)
	}

	sen = <-listen.Chan()
	if sen != nil {
		t.Fatalf("Listen() channel should be closed after EOF; got %#q", sen)
	}
	err = listen.Err()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunAsync(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()
	c.Async()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @r1 []")
		s.writeSentence(t, "!re", ".tag=r1", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done", ".tag=r1")
	}()

	sen, err := c.Run("/ip/address")
	if err != nil {
		t.Fatal(err)
	}
	want := "!re @r1 [{`address` `1.2.3.4/32`}]\n!done @r1 []"
	if sen.String() != want {
		t.Fatalf("/ip/address (%s); want (%s)", sen, want)
	}
}

func TestRunEmptySentence(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t)
		s.writeSentence(t, "!re", "=address=1.2.3.4/32")
		s.writeSentence(t, "!done")
	}()

	sen, err := c.Run("/ip/address")
	if err != nil {
		t.Fatal(err)
	}
	want := "!re @ [{`address` `1.2.3.4/32`}]\n!done @ []"
	if sen.String() != want {
		t.Fatalf("/ip/address (%s); want (%s)", sen, want)
	}
}

func TestRunEOF(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @ []")
	}()

	_, err := c.Run("/ip/address")
	if err == nil {
		t.Fatalf("Run succeeded; want error")
	}
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestRunEOFAsync(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()
	c.Async()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @r1 []")
		s.writeSentence(t, "!re", "=address=1.2.3.4/32")
	}()

	_, err := c.Run("/ip/address")
	if err == nil {
		t.Fatalf("Run succeeded; want error")
	}
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestRunInvalidSentence(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!xxx")
	}()

	_, err := c.Run("/ip/address")
	if err == nil {
		t.Fatalf("Run succeeded; want error")
	}
	if err.Error() != "unknown RouterOS reply word: !xxx" {
		t.Fatal(err)
	}
}

func TestRunTrap(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!trap", "=message=Some device error message")
	}()

	_, err := c.Run("/ip/address")
	if err == nil {
		t.Fatalf("Run succeeded; want error")
	}
	if err.Error() != "from RouterOS device: Some device error message" {
		t.Fatal(err)
	}
}

func TestRunMesagelessTrap(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!trap", "=some=unknown key")
	}()

	_, err := c.Run("/ip/address")
	if err == nil {
		t.Fatalf("Run succeeded; want error")
	}
	if err.Error() != "from RouterOS device: unknown error: !trap @ [{`some` `unknown key`}]" {
		t.Fatal(err)
	}
}

func TestRunFatal(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address @ []")
		s.writeSentence(t, "!fatal", "=message=Some device error message")
	}()

	_, err := c.Run("/ip/address")
	if err == nil {
		t.Fatalf("Run succeeded; want error")
	}
	if err.Error() != "from RouterOS device: Some device error message" {
		t.Fatal(err)
	}
}

func TestRunAfterClose(t *testing.T) {
	c, s := newPair(t)
	c.Close()
	s.Close()

	_, err := c.Run("/ip/address")
	if err == nil {
		t.Fatalf("Run succeeded; want error")
	}
	if err.Error() != "io: read/write on closed pipe" {
		t.Fatal(err)
	}
}

func TestListen(t *testing.T) {
	c, s := newPair(t)
	defer c.Close()

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/address/listen @l1 []")
		s.writeSentence(t, "!re", ".tag=l1", "=address=1.2.3.4/32")
		s.readSentence(t, "/cancel @r2 [{`tag` `l1`}]")
		s.writeSentence(t, "!trap", "=category=2", ".tag=l1")
		s.writeSentence(t, "!done", "=tag=r2")
		s.writeSentence(t, "!done", "=tag=l1")
	}()

	c.Queue = 1
	listen, err := c.Listen("/ip/address/listen")
	if err != nil {
		t.Fatal(err)
	}
	reC := listen.Chan()

	listen.Cancel()

	sen := <-reC
	want := "!re @l1 [{`address` `1.2.3.4/32`}]"
	if fmt.Sprintf("%s", sen) != want {
		t.Fatalf("/ip/address/listen (%s); want (%s)", sen, want)
	}

	sen = <-reC
	if sen != nil {
		t.Fatalf("Listen() channel should be closed after Close(); got %#q", sen)
	}
	err = listen.Err()
	if err != nil {
		t.Fatal(err)
	}
}

type conn struct {
	*io.PipeReader
	*io.PipeWriter
}

func (c *conn) Close() error {
	c.PipeReader.Close()
	c.PipeWriter.Close()
	return nil
}

func newPair(t *testing.T) (*routeros.Client, *fakeServer) {
	ar, aw := io.Pipe()
	br, bw := io.Pipe()

	c, err := routeros.NewClient(&conn{ar, bw})
	if err != nil {
		t.Fatal(err)
	}

	s := &fakeServer{
		proto.NewReader(br),
		proto.NewWriter(aw),
		&conn{br, aw},
	}

	return c, s
}

type fakeServer struct {
	r proto.Reader
	w proto.Writer
	io.Closer
}

func (f *fakeServer) readSentence(t *testing.T, want string) {
	sen, err := f.r.ReadSentence()
	if err != nil {
		t.Fatal(err)
	}
	if sen.String() != want {
		t.Fatalf("Sentence (%s); want (%s)", sen.String(), want)
	}
	t.Logf("< %s\n", sen)
}

func (f *fakeServer) writeSentence(t *testing.T, sentence ...string) {
	t.Logf("> %#q\n", sentence)
	f.w.BeginSentence()
	for _, word := range sentence {
		f.w.WriteWord(word)
	}
	err := f.w.EndSentence()
	if err != nil {
		t.Fatal(err)
	}
}
