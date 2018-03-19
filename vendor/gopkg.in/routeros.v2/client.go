/*
Package routeros is a pure Go client library for accessing Mikrotik devices using the RouterOS API.
*/
package routeros

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"gopkg.in/routeros.v2/proto"
)

// Client is a RouterOS API client.
type Client struct {
	Queue int

	rwc     io.ReadWriteCloser
	r       proto.Reader
	w       proto.Writer
	closing bool
	async   bool
	nextTag int64
	tags    map[string]sentenceProcessor
	mu      sync.Mutex
}

// NewClient returns a new Client over rwc. Login must be called.
func NewClient(rwc io.ReadWriteCloser) (*Client, error) {
	return &Client{
		rwc: rwc,
		r:   proto.NewReader(rwc),
		w:   proto.NewWriter(rwc),
	}, nil
}

// Dial connects and logs in to a RouterOS device.
func Dial(address, username, password string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return newClientAndLogin(conn, address, username, password)
}

// Dial connects and logs in to a RouterOS device.
func DialTimeout(address, username, password string, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, err
	}
	return newClientAndLogin(conn, address, username, password)
}

// DialTLS connects and logs in to a RouterOS device using TLS.
func DialTLS(address, username, password string, tlsConfig *tls.Config) (*Client, error) {
	conn, err := tls.Dial("tcp", address, tlsConfig)
	if err != nil {
		return nil, err
	}
	return newClientAndLogin(conn, address, username, password)
}

// DialTLSTimeout connects and logs in to a RouterOS device using TLS with timeout.
func DialTLSTimeout(address, username, password string, tlsConfig *tls.Config, timeout time.Duration) (*Client, error) {
	dialer := new(net.Dialer)
	dialer.Timeout = timeout

	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return nil, err
	}
	return newClientAndLogin(conn, address, username, password)
}

func newClientAndLogin(rwc io.ReadWriteCloser, address, username, password string) (*Client, error) {
	c, err := NewClient(rwc)
	if err != nil {
		rwc.Close()
		return nil, err
	}
	err = c.Login(username, password)
	if err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// Close closes the connection to the RouterOS device.
func (c *Client) Close() {
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		return
	}
	c.closing = true
	c.mu.Unlock()
	c.rwc.Close()
}

// Login runs the /login command. Dial and DialTLS call this automatically.
func (c *Client) Login(username, password string) error {
	r, err := c.Run("/login")
	if err != nil {
		return err
	}
	ret, ok := r.Done.Map["ret"]
	if !ok {
		return errors.New("RouterOS: /login: no ret (challenge) received")
	}
	b, err := hex.DecodeString(ret)
	if err != nil {
		return fmt.Errorf("RouterOS: /login: invalid ret (challenge) hex string received: %s", err)
	}

	r, err = c.Run("/login", "=name="+username, "=response="+c.challengeResponse(b, password))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) challengeResponse(cha []byte, password string) string {
	h := md5.New()
	h.Write([]byte{0})
	io.WriteString(h, password)
	h.Write(cha)
	return fmt.Sprintf("00%x", h.Sum(nil))
}
