package audio

import "time"

// SampleRate is the capture rate used for all recordings, in samples per second.
const SampleRate = 16000

// Clip is a recording held in memory: mono, 16-bit PCM samples.
type Clip struct {
	Samples    []int16
	SampleRate int
}

// Duration returns the length of the recording.
func (c Clip) Duration() time.Duration {
	if c.SampleRate <= 0 {
		return 0
	}
	return time.Duration(len(c.Samples)) * time.Second / time.Duration(c.SampleRate)
}
