package eval_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/MadSP-McDaniel/eipsim/util"
)

func TestFourierFig(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	out, _ := os.Create("./figs/fourier.csv")
	var fs []util.Fourier
	for i := 0; i < 10; i++ {
		fs = append(fs, util.RandomFourier(r, 24))
	}
	for i := 0.0; i < 1.0; i += 1.0 / 24 / 4 {
		for j, f := range fs {
			fmt.Fprintf(out, "%d,%f,%f\n", j, i, 50+50*f.Compute(i))
		}
	}
	out.Close()
}
