package eval

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/agents"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/policies"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/simulator"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"
)

func NSP() types.PoolPolicy {
	return policies.NewSegmentedPool(1, false)
}

var poolMakers = []func() types.PoolPolicy{policies.NewFIFOPool, policies.NewRandomPool, policies.NewTaggedPool, NSP}

// var poolMakers = []func() types.PoolPolicy{NSP}

func TestBenign(t *testing.T) {
	simulators := make(chan *simulator.Simulator)

	rp := policies.NewRandomPool()
	s := simulator.NewSimulator(2000000, rp, 1)
	s.AllocationSamplingRate = 100
	s.RegisterStatCollector(func(s types.Simulator, m map[string]interface{}) {
		log.Println(s.GetTime())
		// f, err := os.Create("benign.prof")
		// if err != nil {
		// 	panic(err)
		// }
		// defer f.Close()
		// err = pprof.WriteHeapProfile(f)
		// if err != nil {
		// 	panic(err)
		// }
	})
	s.MaxTime = 10 * types.Day
	s.AddAgent(&agents.AutoscaleAgent{NumTenants: 120000, MaxWait: 600, NMax: 30, NMin: 2, BaseAgent: agents.BaseAgent{Type: "autoscale"}, TenantChurn: 365 * types.Day})
	s.StatCollectionInterval = 1 * types.Hour
	s.LatentConfProbability = LatentConfProbability
	s.ProcessAll()

	MaxUsedIPsWithTimeout := 2000000 - rp.(*policies.RandomPool).MinAvailable

	for _, pool := range poolMakers {
		for _, allocRatio := range allocRatios {
			ar := allocRatio
			p := pool()
			t.Run(fmt.Sprintf("%s %d", p.GetType(), ar), func(t *testing.T) {
				s := simulator.NewSimulator(MaxUsedIPsWithTimeout*100/ar, p, 1)
				s.RegisterStatCollector(func(s types.Simulator, m map[string]interface{}) {
					// log.Println(s.GetTime())
					// if isLead {
					// 	f, err := os.Create("benign.prof")
					// 	if err != nil {
					// 		panic(err)
					// 	}
					// 	defer f.Close()
					// 	err = pprof.WriteHeapProfile(f)
					// 	if err != nil {
					// 		panic(err)
					// 	}
					// }
				})
				s.StatCollectionInterval = 1 * types.Hour
				s.AllocationSamplingRate = 100
				t.Parallel()
				s.MaxTime = 180 * types.Day
				s.LatentConfProbability = LatentConfProbability
				s.AddAgent(&agents.AutoscaleAgent{NumTenants: 120000, MaxWait: 600, NMax: 30, NMin: 2, BaseAgent: agents.BaseAgent{Type: "autoscale"}, TenantChurn: 365 * types.Day})
				s.ProcessAll()
				s.OverallStats["targetAllocRatio"] = ar
				simulators <- s
			})
		}
	}

	done := make(chan struct{})

	t.Cleanup(func() {
		close(simulators)
		<-done
	})
	go writeSims("./figs/syn-benign.jsonl", simulators, done)
}

func writeSims(f string, sims chan *simulator.Simulator, donechan chan struct{}) {
	out, err := os.Create(f)
	if err != nil {
		panic(err)
	}
	for sim := range sims {
		data, err := json.Marshal(sim)
		if err != nil {
			panic(err)
		}
		_, err = out.Write(data)
		if err != nil {
			panic(err)
		}
		_, err = out.WriteString("\n")
		if err != nil {
			panic(err)
		}
	}
	out.Close()
	close(donechan)
}
