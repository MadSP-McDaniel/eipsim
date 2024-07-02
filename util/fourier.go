package util

import (
	"math"
	"math/rand"
)

type Fourier struct {
	amplitudes []float64
	phases     []float64
}

func RandomFourier(r *rand.Rand, dims int) Fourier {
	f := Fourier{}
	for i := 0; i < dims; i++ {
		f.amplitudes = append(f.amplitudes, r.Float64())
		f.phases = append(f.phases, r.Float64())
	}
	f.phases[0] /= 2
	return f
}

func (f Fourier) Compute(x float64) float64 {
	var result float64
	var max float64
	for i := range f.amplitudes {
		n := float64(1 + i)
		max += 1 / n
		result += f.amplitudes[i] * math.Sin(n*2*math.Pi*(x+f.phases[i])) / n
	}
	result = result/max + 0.5
	if result < 0 {
		result = 0
	}
	if result > 1 {
		result = 1
	}
	return result
}
