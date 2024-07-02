package util

import (
	"math"
	"math/rand"
)

func SampleExponential(r *rand.Rand, lambda float64) float64 {
	return -math.Log(1-r.Float64()) / lambda
}
