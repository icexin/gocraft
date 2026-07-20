package main

import "sync"

const (
	SampleRate = 44100
)

type Voice struct {
	generator SoundGenerator
	elapsed   float64
	duration  float64
	volume    float32
	done      bool
}

type Mixer struct {
	mu     sync.Mutex
	voices []*Voice
}

func NewMixer() *Mixer {
	return &Mixer{}
}

func (m *Mixer) Play(gen SoundGenerator, duration float64, volume float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.voices = append(m.voices, &Voice{
		generator: gen,
		duration:  duration,
		volume:    volume,
	})
}

func (m *Mixer) ReadSamples(n int) []float32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]float32, n)
	dt := 1.0 / float64(SampleRate)
	for i := range out {
		var sample float32
		for _, v := range m.voices {
			if v.done {
				continue
			}
			s := v.generator.Sample(v.elapsed)
			sample += s * v.volume
			v.elapsed += dt
			if v.elapsed >= v.duration {
				v.done = true
			}
		}
		if sample > 1.0 {
			sample = 1.0
		}
		if sample < -1.0 {
			sample = -1.0
		}
		out[i] = sample
	}
	active := m.voices[:0]
	for _, v := range m.voices {
		if !v.done {
			active = append(active, v)
		}
	}
	m.voices = active
	return out
}

func (m *Mixer) ActiveVoices() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.voices)
}
