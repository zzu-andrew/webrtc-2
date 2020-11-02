package webrtc

import (
	"context"
	"errors"
	"io"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

type Interceptor interface {
	Intercept(*PeerConnection, ReadWriter) ReadWriter
}

// Reader is an interface to handle incoming RTP stream.
type ReadWriter interface {
	ReadRTP(context.Context) (*rtp.Packet, map[interface{}]interface{}, error)
	WriteRTP(context.Context, *rtp.Packet, map[interface{}]interface{}) error
	ReadRTCP(context.Context) ([]rtcp.Packet, error)
	WriteRTCP(context.Context, []rtcp.Packet) error
	io.Closer
}

type contextReadWriter struct{}

type interceptorChain struct {
	readWriter ReadWriter
}

type keyReadRTP struct{}
type keyReadRTCP struct{}
type keyWriteRTP struct{}
type keyWriteRTCP struct{}

type writeRTP func(packet *rtp.Packet)
type writeRTCP func(packets []rtcp.Packet)

func (c *contextReadWriter) ReadRTP(ctx context.Context) (*rtp.Packet, map[interface{}]interface{}, error) {
	p, ok := ctx.Value(keyReadRTP{}).(*rtp.Packet)
	if !ok {
		return nil, nil, errors.New("packet not found in context")
	}

	return p, make(map[interface{}]interface{}), nil
}

func (c *contextReadWriter) WriteRTP(ctx context.Context, packet *rtp.Packet, _ map[interface{}]interface{}) error {
	writeRTP, ok := ctx.Value(keyWriteRTP{}).(writeRTP)
	if !ok {
		return errors.New("callback not found in context")
	}
	writeRTP(packet)

	return nil
}

func (c *contextReadWriter) ReadRTCP(ctx context.Context) ([]rtcp.Packet, error) {
	p, ok := ctx.Value(keyReadRTCP{}).([]rtcp.Packet)
	if !ok {
		return nil, errors.New("packets not found in context")
	}
	return p, nil
}

func (c *contextReadWriter) WriteRTCP(ctx context.Context, packets []rtcp.Packet) error {
	writeRTCP, ok := ctx.Value(keyWriteRTCP{}).(writeRTCP)
	if !ok {
		return errors.New("callback not found in context")
	}
	writeRTCP(packets)

	return nil
}

func (c *contextReadWriter) Close() error {
	return nil
}

func newInterceptorChain(pc *PeerConnection, interceptors []Interceptor) *interceptorChain {
	var readWriter ReadWriter = &contextReadWriter{}
	for _, interceptor := range interceptors {
		readWriter = interceptor.Intercept(pc, readWriter)
	}
	return &interceptorChain{readWriter: readWriter}
}

func (i *interceptorChain) wrapReadRTP(packet *rtp.Packet) (*rtp.Packet, error) {
	ctx := context.WithValue(context.Background(), keyReadRTP{}, packet)
	p, _, err := i.readWriter.ReadRTP(ctx)
	return p, err
}

func (i *interceptorChain) wrapWriteRTP(packet *rtp.Packet) (*rtp.Packet, error) {
	var p *rtp.Packet
	ctx := context.WithValue(context.Background(), keyWriteRTP{}, func(p2 *rtp.Packet) {
		p = p2
	})
	err := i.readWriter.WriteRTP(ctx, packet, make(map[interface{}]interface{}))
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (i *interceptorChain) wrapReadRTCP(packets []rtcp.Packet) ([]rtcp.Packet, error) {
	ctx := context.WithValue(context.Background(), keyReadRTCP{}, packets)
	return i.readWriter.ReadRTCP(ctx)
}

func (i *interceptorChain) wrapWriteRTCP(packet *rtp.Packet) (*rtp.Packet, error) {
	var p *rtp.Packet
	ctx := context.WithValue(context.Background(), keyWriteRTP{}, func(p2 *rtp.Packet) {
		p = p2
	})
	err := i.readWriter.WriteRTP(ctx, packet, make(map[interface{}]interface{}))
	if err != nil {
		return nil, err
	}

	return p, nil
}
