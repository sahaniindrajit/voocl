package audio

import (
	"testing"
	"time"
)

func TestClipDuration(t *testing.T) {
	tests := []struct {
		name string
		clip Clip
		want time.Duration
	}{
		{"empty", Clip{SampleRate: SampleRate}, 0},
		{"one second", Clip{Samples: make([]int16, SampleRate), SampleRate: SampleRate}, time.Second},
		{"half a second", Clip{Samples: make([]int16, SampleRate/2), SampleRate: SampleRate}, 500 * time.Millisecond},
		{"one sample", Clip{Samples: make([]int16, 1), SampleRate: SampleRate}, time.Second / SampleRate},
		{"other sample rate", Clip{Samples: make([]int16, 8000), SampleRate: 8000}, time.Second},
		{"zero sample rate", Clip{Samples: make([]int16, 100)}, 0},
		{"negative sample rate", Clip{Samples: make([]int16, 100), SampleRate: -1}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.clip.Duration(); got != tt.want {
				t.Errorf("Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}
