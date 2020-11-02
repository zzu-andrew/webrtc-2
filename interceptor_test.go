package webrtc

import (
	"context"
	"testing"

	"github.com/pion/rtp"
)

type testInterceptor1 struct {
}

type testInterceptor2 struct {
	t *testing.T
}

type testReadWriter1 struct {
	ReadWriter
}

type testReadWriter2 struct {
	ReadWriter
	t *testing.T
}

type testInterceptorKey struct{}

func (t *testInterceptor1) Intercept(_ *PeerConnection, readWriter ReadWriter) ReadWriter {
	return &testReadWriter1{ReadWriter: readWriter}
}

func (t *testInterceptor2) Intercept(_ *PeerConnection, readWriter ReadWriter) ReadWriter {
	return &testReadWriter2{ReadWriter: readWriter, t: t.t}
}

func (t *testReadWriter1) ReadRTP(ctx context.Context) (*rtp.Packet, map[interface{}]interface{}, error) {
	p, m, err := t.ReadWriter.ReadRTP(ctx)
	if err != nil {
		return nil, nil, err
	}

	p.SSRC = 1
	m[testInterceptorKey{}] = "read1"

	return p, m, nil
}

func (t *testReadWriter1) WriteRTP(ctx context.Context, p *rtp.Packet, m map[interface{}]interface{}) error {
	p.SSRC = 1
	m[testInterceptorKey{}] = "read1"

	return t.ReadWriter.WriteRTP(ctx, p, m)
}

func (t *testReadWriter2) ReadRTP(ctx context.Context) (*rtp.Packet, map[interface{}]interface{}, error) {
	p, m, err := t.ReadWriter.ReadRTP(ctx)
	if err != nil {
		return nil, nil, err
	}

	if p.SSRC != 1 {
		t.t.Errorf("expected SSRC to be 1, got: %d", p.SSRC)
	}
	metaVal := m[testInterceptorKey{}]
	if metaVal != "read1" {
		t.t.Errorf("expected meta to be set to read1, got: %s", metaVal)
	}

	// test replacing the packet
	p2 := &rtp.Packet{}
	p2.SSRC = 2

	return p2, m, nil
}

func (t *testReadWriter2) WriteRTP(ctx context.Context, p *rtp.Packet, m map[interface{}]interface{}) error {
	if p.SSRC != 1 {
		t.t.Errorf("expected SSRC to be 1, got: %d", p.SSRC)
	}
	metaVal := m[testInterceptorKey{}]
	if metaVal != "read1" {
		t.t.Errorf("expected meta to be set to read1, got: %s", metaVal)
	}

	// test replacing the packet
	p2 := &rtp.Packet{}
	p2.SSRC = 2

	return t.ReadWriter.WriteRTP(ctx, p2, m)
}

func TestInterceptorChainReadRTP(t *testing.T) {
	chain := newInterceptorChain(nil, []Interceptor{
		&testInterceptor1{},
		&testInterceptor2{t: t},
	})

	p, err := chain.wrapReadRTP(&rtp.Packet{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if p.SSRC != 2 {
		t.Errorf("expected SSRC to be 2, got: %d", p.SSRC)
	}
}

func TestInterceptorChainWriteRTP(t *testing.T) {
	chain := newInterceptorChain(nil, []Interceptor{
		&testInterceptor1{},
		&testInterceptor2{t: t},
	})

	p, err := chain.wrapWriteRTP(&rtp.Packet{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if p.SSRC != 2 {
		t.Errorf("expected SSRC to be 2, got: %d", p.SSRC)
	}
}
