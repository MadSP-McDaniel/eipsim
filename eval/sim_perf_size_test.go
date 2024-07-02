package eval

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/agents"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/simulator"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"
)

func junkRound(f float64) float64 {
	if f > 100 {
		return math.Round(f/10) * 10
	}
	if f > 10 {
		return math.Round(f)
	}
	return math.Round(f*10) / 10
}

func SIify(i float64, unit string) string {
	if i < 1 {
		return fmt.Sprintf("\\SI{%2g}{m%s}", junkRound(float64(i)*1000), unit)
	}
	if i < 100 {
		return fmt.Sprintf("\\SI{%2g}{%s}", junkRound(float64(i)), unit)
	}
	if i < 1000 {
		return fmt.Sprintf("\\SI{%d}{%s}", int(i), unit)
	}
	if i < 1000000 {
		return fmt.Sprintf("\\SI{%2g}{k%s}", junkRound(float64(i)/1000), unit)
	}
	if i < 1000000000 {
		return fmt.Sprintf("\\SI{%2g}{M%s}", junkRound(float64(i)/1000000), unit)
	}
	panic("Too big!")
}

func BenchmarkSimPerfSizeTest(b *testing.B) {
	const numDays = 100

	f, err := os.Create("./figs/perf-100d.tex")
	if err != nil {
		panic(err)
	}

	for size := 100; size <= 10000000; size *= 10 {
		start := time.Now()
		s := simulator.NewSimulator(size, NSP(), 1)
		s.MaxTime = numDays * types.Day
		s.AddAgent(&agents.AutoscaleAgent{NumTenants: size / 10, MaxWait: 600, NMax: 10, NMin: 1, BaseAgent: agents.BaseAgent{Type: "autoscale"}, TenantChurn: 365 * types.Day})
		s.AllocationSamplingRate = 1000000
		// adversary := &agents.AdversarialAgent{
		// 	MaxCreated:   500000,
		// 	MaxIPs:       60,
		// 	HoldDuration: 10 * types.Minute,
		// 	MaxPerCycle:  10,
		// 	StartTime:    10000000 * types.Day,
		// }
		// s.AddAgent(adversary)
		s.ProcessAll()
		d := time.Since(start)

		fmt.Fprintf(f, "%s & %s & %s & %s & %s \\\\ \n",
			SIify(float64(size), ""),
			SIify(d.Seconds(), "s"),
			SIify(float64(numDays*types.Day)/d.Seconds(), ""),
			SIify(float64(s.GetAllocated()), ""),
			SIify(float64(s.GetAllocated())/d.Seconds(),
				""))
	}

	f.Close()
}
