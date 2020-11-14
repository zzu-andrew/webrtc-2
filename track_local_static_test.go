package webrtc

import (
	"testing"
	"time"

	"github.com/pion/transport/test"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sean-Der
// * Writing to a un-added Track
// * Write to a track on Closed PeerConnection
// * Write to a track on a disconnected PeerConnection
// * Write to a track with one bad PeerConnection
// * Offer fail when Answerer doesn't support a codec
// * Answer fail when Offerer doesn't support a codec
// * Does sending use the proper Payload type

func TestTrackLocalStatic_WriteToUnAddedTrack(t *testing.T) {
	track, err := NewTrackLocalStaticSample(RTPCodecCapability{MimeType: "video/vp8"}, "foo", "bar")
	if err != nil {
		panic(err)
	}

	err = track.WriteSample(media.Sample{Data: []byte{0x00}, Duration: time.Second})
	assert.Nil(t, err)
}

func TestTrackLocalStatic_WriteToClosedPeerConnection(t *testing.T) {
	lim := test.TimeOut(time.Second * 30)
	defer lim.Stop()

	report := test.CheckRoutines(t)
	defer report()

	pca, pcb, err := newPair()
	require.NoError(t, err)

	defer pcb.Close()

	track, err := NewTrackLocalStaticSample(RTPCodecCapability{MimeType: "video/vp8"}, "foo", "bar")
	require.NoError(t, err)

	_, err = pca.AddTransceiverFromTrack(track)
	require.NoError(t, err)

	connected := make(chan struct{})
	disconnected := make(chan struct{})

	pca.OnICEConnectionStateChange(func(state ICEConnectionState) {
		if state == ICEConnectionStateConnected {
			close(connected)
		}

		if state == ICEConnectionStateClosed {
			close(disconnected)
		}
	})

	err = signalPair(pca, pcb)
	require.NoError(t, err)

	select {
	case <-connected:
	case <-time.After(time.Second):
		require.Fail(t, "timed out while waiting to connect")
	}

	err = pca.Close()
	require.NoError(t, err)

	select {
	case <-disconnected:
	case <-time.After(time.Second):
		require.Fail(t, "timed out while waiting to connect")
	}

	err = track.WriteSample(media.Sample{Data: []byte{0x00}, Duration: time.Second})
	assert.Nil(t, err)
}

func TestTrackLocalStatic_OneBadPeerConnection(t *testing.T) {
	// TODO not sure what "bad" means.
}

func TestTrackLocalStatic_OfferFailWhenAnswererDoesNotSupportCodec(t *testing.T) {
	lim := test.TimeOut(time.Second * 30)
	defer lim.Stop()

	report := test.CheckRoutines(t)
	defer report()

	mediaEngine1 := MediaEngine{}
	err := mediaEngine1.RegisterCodec(RTPCodecParameters{
		RTPCodecCapability: RTPCodecCapability{mimeTypeVP8, 90000, 0, "", nil},
		PayloadType:        96,
	}, RTPCodecTypeVideo)
	require.NoError(t, err)

	api1 := NewAPI(WithMediaEngine(mediaEngine1))
	pca, err := api1.NewPeerConnection(Configuration{})
	require.NoError(t, err)

	defer pca.Close()

	mediaEngine2 := MediaEngine{}
	err = mediaEngine2.RegisterCodec(RTPCodecParameters{
		RTPCodecCapability: RTPCodecCapability{mimeTypeVP9, 90000, 0, "profile=0", nil},
		PayloadType:        96,
	}, RTPCodecTypeVideo)
	require.NoError(t, err)

	api2 := NewAPI(WithMediaEngine(mediaEngine2))
	pcb, err := api2.NewPeerConnection(Configuration{})
	require.NoError(t, err)

	_, err = pcb.AddTransceiverFromKind(RTPCodecTypeVideo)
	require.NoError(t, err)

	defer pcb.Close()

	track, err := NewTrackLocalStaticSample(RTPCodecCapability{MimeType: mimeTypeVP8}, "foo", "bar")
	require.NoError(t, err)

	_, err = pca.AddTransceiverFromTrack(track)
	require.NoError(t, err)

	connected := make(chan struct{})

	pca.OnICEConnectionStateChange(func(state ICEConnectionState) {
		if state == ICEConnectionStateConnected {
			close(connected)
		}
	})

	err = signalPair(pca, pcb)
	require.NoError(t, err)
	// require.EqualError(t, err, "bla")

	select {
	case <-connected:
	case <-time.After(time.Second):
		require.Fail(t, "timed out while waiting to connect")
	}
}
