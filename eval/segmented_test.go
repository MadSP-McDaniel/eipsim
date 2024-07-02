package eval

import (
	"fmt"
	"testing"

	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/agents"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/policies"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/simulator"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"
)

func TestSegmentedPoolSize(t *testing.T) {
	simulators := make(chan *simulator.Simulator)

	s := simulator.NewSimulator(1000000, policies.NewRandomPool(), 1)
	s.MaxTime = 10 * types.Day
	s.AddAgent(&agents.AutoscaleAgent{NumTenants: 120000, MaxWait: 600, NMax: 30, NMin: 2, BaseAgent: agents.BaseAgent{Type: "autoscale"}, TenantChurn: 365 * types.Day})
	s.ProcessAll()
	MaxUsedIPs := s.MaxUsedIPs

	for multiplier := 0.0; multiplier <= 5.0; multiplier += 0.05 {
		m := multiplier
		t.Run(fmt.Sprintf("%f", m), func(t *testing.T) {
			pool := policies.SegmentedPool{TimerMultiplier: float64(m)}
			s := simulator.NewSimulator(MaxUsedIPs*100/95, &pool, 1)
			s.StatCollectionInterval = 1 * types.Hour
			s.AllocationSamplingRate = 100
			t.Parallel()
			s.MaxTime = 210 * types.Day
			s.LatentConfProbability = LatentConfProbability
			s.AddAgent(&agents.AutoscaleAgent{NumTenants: 120000, MaxWait: 600, NMax: 30, NMin: 2, BaseAgent: agents.BaseAgent{Type: "autoscale"}, TenantChurn: 365 * types.Day})
			s.AddAgent(&agents.AdversarialAgent{
				MaxCreated:           50000000,
				MaxIPs:               60,
				HoldDuration:         10 * types.Minute,
				MaxPerCycle:          10,
				StartTime:            180 * types.Day,
				AllocationsPerTenant: 60,
				MaxTenants:           10000000,
				SegmentedPool:        &pool,
				BaseAgent:            agents.BaseAgent{Type: "adversary"},
			})
			s.ProcessAll()
			s.OverallStats["multiplier"] = m
			simulators <- s
		})
	}

	done := make(chan struct{})

	t.Cleanup(func() {
		close(simulators)
		<-done
	})
	go writeSims("./figs/segmented_multipliers.jsonl", simulators, done)
}
