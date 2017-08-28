package routeros

import (
	"bytes"
	"fmt"

	"gopkg.in/routeros.v2/proto"
)

// Reply has all the sentences from a reply.
type Reply struct {
	Re   []*proto.Sentence
	Done *proto.Sentence
}

func (r *Reply) String() string {
	b := &bytes.Buffer{}
	for _, re := range r.Re {
		fmt.Fprintf(b, "%s\n", re)
	}
	fmt.Fprintf(b, "%s", r.Done)
	return b.String()
}

// readReply reads one reply synchronously. It returns the reply.
func (c *Client) readReply() (*Reply, error) {
	r := &Reply{}
	for {
		sen, err := c.r.ReadSentence()
		if err != nil {
			return nil, err
		}
		done, err := r.processSentence(sen)
		if err != nil {
			return nil, err
		}
		if done {
			return r, nil
		}
	}
}

func (r *Reply) processSentence(sen *proto.Sentence) (bool, error) {
	switch sen.Word {
	case "!re":
		r.Re = append(r.Re, sen)
	case "!done":
		r.Done = sen
		return true, nil
	case "!trap", "!fatal":
		return true, &DeviceError{sen}
	case "":
		// API docs say that empty sentences should be ignored
	default:
		return true, &UnknownReplyError{sen}
	}
	return false, nil
}
