package collector

import (
	"io"
	"mikrotik-exporter/config"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	routeros "gopkg.in/routeros.v2"
	"gopkg.in/routeros.v2/proto"
)

func TestCloudMetricDesc(t *testing.T) {
	c := getFakeClient(t, "false")
	defer c.Close()

	cloudCollector := newCloudCollector()
	metrics := make(chan prometheus.Metric, 1)

	ctx := collectorContext{
		ch:     metrics,
		device: &config.Device{Name: "foo", Address: "test"},
		client: c,
	}

	cloudCollector.collect(&ctx)
	m := <-metrics

	d := descriptionForPropertyName("cloud", "ddns-enabled", []string{"name", "address", "public_address"})

	assert.Equal(t, d, m.Desc(), "metrics description missmatch")
}

func TestCloudCollectFalse(t *testing.T) {
	var pb dto.Metric

	c := getFakeClient(t, "false")
	defer c.Close()

	cloudCollector := newCloudCollector()
	metrics := make(chan prometheus.Metric, 1)

	ctx := collectorContext{
		ch:     metrics,
		device: &config.Device{Name: "foo", Address: "test"},
		client: c,
	}

	cloudCollector.collect(&ctx)
	m := <-metrics

	assert.NoError(t, nil, m.Write(&pb), "error reading metrics")
	assert.Equal(t, float64(0), pb.Counter.GetValue(), "excpeted output should be 0 for false")

	for _, l := range pb.Label {
		switch l.GetName() {
		case "name":
			assert.Equal(t, "foo", l.GetValue(), "device name label value missmatch")
		case "address":
			assert.Equal(t, "test", l.GetValue(), "device address label value missmatch")
		case "public_address":
			assert.Equal(t, "0.0.0.0", l.GetValue(), "public_address label value missmatch")
		default:
			t.Fatalf("invalid or missing lables %s", l.GetName())
		}
	}
}

func TestCloudCollectTrue(t *testing.T) {
	var pb dto.Metric

	c := getFakeClient(t, "true")
	defer c.Close()

	cloudCollector := newCloudCollector()
	metrics := make(chan prometheus.Metric, 1)

	ctx := collectorContext{
		ch:     metrics,
		device: &config.Device{Name: "foo", Address: "test"},
		client: c,
	}

	cloudCollector.collect(&ctx)
	m := <-metrics

	assert.NoError(t, nil, m.Write(&pb))
	assert.Equal(t, float64(1), pb.Counter.GetValue(), "excpeted output should be 1 for true")
}

func getFakeClient(t *testing.T, state string) *routeros.Client {
	c, s := newPair(t)

	go func() {
		defer s.Close()
		s.readSentence(t, "/ip/cloud/print @ [{`.proplist` `public-address,ddns-enabled`}]")
		s.writeSentence(t, "!re", "=ddns-enabled="+state, "=public-address=0.0.0.0")
		s.writeSentence(t, "!done")
	}()

	return c
}

// Heplers
type fakeServer struct {
	r proto.Reader
	w proto.Writer
	io.Closer
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
