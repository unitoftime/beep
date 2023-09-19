package generators

import (
	"errors"
	"math"

	"github.com/unitoftime/beep"
)

type triangleGenerator struct {
	dt float64
	t  float64
}

// Creates a streamer which will procude an infinite triangle wave with the given frequency.
// use other wrappers of this package to change amplitude or add time limit.
// sampleRate must be at least two times grater then frequency, otherwise this function will return an error.
func TriangleTone(sr beep.SampleRate, freq float64) (beep.Streamer, error) {
	dt := freq / float64(sr)

	if dt >= 1.0/2.0 {
		return nil, errors.New("unitoftime triangle tone generator: samplerate must be at least 2 times grater then frequency")
	}

	return &triangleGenerator{dt, 0}, nil
}

func (g *triangleGenerator) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		if g.t < 0.5 {
			samples[i][0] = 2.0*(1-g.t) - 1
			samples[i][1] = 2.0*(1-g.t) - 1
		} else {
			samples[i][0] = 2.0*g.t - 1.0
			samples[i][1] = 2.0*g.t - 1.0
		}
		_, g.t = math.Modf(g.t + g.dt)
	}

	return len(samples), true
}

func (*triangleGenerator) Err() error {
	return nil
}
