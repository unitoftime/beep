// Package speaker implements playback of beep.Streamer values through physical speakers.
package speaker

import (
	"fmt"
	"math"
	"sync"

	"github.com/unitoftime/beep"
	"github.com/ebitengine/oto/v3"
	"github.com/pkg/errors"
)

var (
	mu      sync.Mutex
	mixer   beep.Mixer
	context *oto.Context
	player  *oto.Player
)

const channelCount = 2
const bitDepthInBytes = 2
const bytesPerSample = bitDepthInBytes * channelCount

// Init initializes audio playback through speaker. Must be called before using this package.
//
// The bufferSize argument specifies the number of samples of the speaker's buffer. Bigger
// bufferSize means lower CPU usage and more reliable playback. Lower bufferSize means better
// responsiveness and less delay.
func Init(sampleRate beep.SampleRate, bufferSize int) error {
	mu.Lock()
	defer mu.Unlock()

	// Force it to be a multiple of 2
	div := int(math.Log2(float64(bufferSize)))
	bufferSize = int(math.Exp2(float64(div)))

	// Calculate the buffer time to pass into oto (Note we add +1 to ensure they match our value when they floor)
	bufferTime := sampleRate.D(bufferSize + 1)

	op := &oto.NewContextOptions{
		SampleRate: int(sampleRate),
		ChannelCount: 2,
		Format: oto.FormatSignedInt16LE,
		BufferSize: bufferTime,
	}
	var err error
	var readyChan chan struct{}
	context, readyChan, err = oto.NewContext(op)
	if err != nil {
		return errors.Wrap(err, "failed to initialize speaker")
	}

	<- readyChan // TODO: Dont block here

	mixer = beep.Mixer{}
	mainReader := newReader()

	player = context.NewPlayer(mainReader)
	player.SetBufferSize(bufferSize * bytesPerSample)
	player.Play()

	return nil
}

type reader struct {
	samples [][2]float64
}
func newReader() *reader {
	return &reader{
		make([][2]float64, 0),
	}
}
func (r *reader) Read(buf []byte) (n int, err error) {
	if len(buf) % 4 != 0 {
		err = fmt.Errorf("invalid read length, must be 4, buf: %d", len(buf))
		// panic(err)
		return 0, err
	}

	numSamples := len(buf) / 4
	if cap(r.samples) < numSamples {
		r.samples = make([][2]float64, numSamples)
	}
	r.samples = r.samples[:numSamples]

	mu.Lock()
	mixer.Stream(r.samples)
	mu.Unlock()

	for i := range r.samples {
		for c := range r.samples[i] {
			val := r.samples[i][c]
			if val < -1 {
				val = -1
			}
			if val > +1 {
				val = +1
			}
			valInt16 := int16(val * (1<<15 - 1))
			low := byte(valInt16)
			high := byte(valInt16 >> 8)
			buf[i*4+c*2+0] = low
			buf[i*4+c*2+1] = high
		}
	}
	return 4 * numSamples, nil
}

// Close closes the playback and the driver. In most cases, there is certainly no need to call Close
// even when the program doesn't play anymore, because in properly set systems, the default mixer
// handles multiple concurrent processes. It's only when the default device is not a virtual but hardware
// device, that you'll probably want to manually manage the device from your application.
func Close() {
	if player != nil {
		player.Close()
		player = nil
	}
}

// Lock locks the speaker. While locked, speaker won't pull new data from the playing Streamers. Lock
// if you want to modify any currently playing Streamers to avoid race conditions.
//
// Always lock speaker for as little time as possible, to avoid playback glitches.
func Lock() {
	mu.Lock()
}

// Unlock unlocks the speaker. Call after modifying any currently playing Streamer.
func Unlock() {
	mu.Unlock()
}

// Play starts playing all provided Streamers through the speaker.
func Play(s ...beep.Streamer) {
	mu.Lock()
	mixer.Add(s...)
	mu.Unlock()
}

// Clear removes all currently playing Streamers from the speaker.
func Clear() {
	mu.Lock()
	mixer.Clear()
	mu.Unlock()
}
