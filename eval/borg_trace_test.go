package eval

import (
	"fmt"
	"log"
	"testing"

	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/agents"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/policies"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/simulator"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"
)

func TestBorgAdversaries(t *testing.T) {
	simulators := make(chan *simulator.Simulator)

	rp := policies.NewRandomPool()
	s := simulator.NewSimulator(2000000, rp, 1)
	s.AllocationSamplingRate = 100
	s.RegisterStatCollector(func(s types.Simulator, m map[string]interface{}) {
		log.Println(s.GetTime())
	})
	s.AddAgent(&agents.CSVAgent{InputFilename: "./borg_collections_normalized.csv.zst", Zstd: true, BaseAgent: agents.BaseAgent{Type: "csv"}})
	s.AddAgent(&agents.AdversarialAgent{
		MaxCreated:           0,
		MaxIPs:               60,
		HoldDuration:         10 * types.Minute,
		MaxPerCycle:          10,
		StartTime:            10 * types.Day,
		AllocationsPerTenant: 60,
		MaxTenants:           1,
		BaseAgent:            agents.BaseAgent{Type: "adversary"},
	})
	s.StatCollectionInterval = 1 * types.Hour
	s.LatentConfProbability = LatentConfProbability
	s.ProcessAll()
	MaxUsedIPsWithTimeout := 2000000 - rp.(*policies.RandomPool).MinAvailable

	fmt.Println(s.MaxUsedIPs, MaxUsedIPsWithTimeout)

	for _, pool := range poolMakers {
		for ntIndex := 0; ntIndex < len(numTenants); ntIndex++ {
			for _, allocRatio := range allocRatios {
				ar := allocRatio
				nt := numTenants[ntIndex]
				if ar != 95 && nt != 10000 && nt != 1 {
					continue
				}
				p := pool()
				t.Run(fmt.Sprintf("%s %d %d", p.GetType(), ar, nt), func(t *testing.T) {
					t.Parallel()
					s := simulator.NewSimulator(MaxUsedIPsWithTimeout*100/ar, p, 1)
					s.AllocationSamplingRate = 100
					//s.MaxTime = 2 * types.Day
					s.AddAgent(&agents.CSVAgent{InputFilename: "./borg_collections_normalized.csv.zst", Zstd: true, BaseAgent: agents.BaseAgent{Type: "csv"}})
					s.StatCollectionInterval = 1 * types.Hour
					s.LatentConfProbability = LatentConfProbability
					s.AddAgent(&agents.AdversarialAgent{
						MaxCreated:           0,
						MaxIPs:               60,
						HoldDuration:         10 * types.Minute,
						MaxPerCycle:          10,
						StartTime:            10 * types.Day,
						AllocationsPerTenant: 60,
						MaxTenants:           nt,
						BaseAgent:            agents.BaseAgent{Type: "adversary"},
					})
					s.ProcessAll()
					s.OverallStats["targetAllocRatio"] = ar
					s.OverallStats["numAdversaryTenants"] = nt
					simulators <- s
				})
			}
		}

	}
	done := make(chan struct{})

	t.Cleanup(func() {
		close(simulators)
		<-done
	})
	go writeSims("./figs/borg-adv.jsonl", simulators, done)
}

func TestBorg(t *testing.T) {
	simulators := make(chan *simulator.Simulator)

	rp := policies.NewRandomPool()
	s := simulator.NewSimulator(2000000, rp, 1)
	s.AllocationSamplingRate = 100
	s.RegisterStatCollector(func(s types.Simulator, m map[string]interface{}) {
		log.Println(s.GetTime())
	})
	s.AddAgent(&agents.CSVAgent{InputFilename: "./borg_collections_normalized.csv.zst", Zstd: true, BaseAgent: agents.BaseAgent{Type: "csv"}})
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
				s.AllocationSamplingRate = 100
				s.StatCollectionInterval = 1 * types.Hour
				t.Parallel()
				s.MaxTime = 10 * types.Day
				s.LatentConfProbability = LatentConfProbability
				s.AddAgent(&agents.CSVAgent{InputFilename: "./borg_collections_normalized.csv.zst", Zstd: true, BaseAgent: agents.BaseAgent{Type: "csv"}})
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
	go writeSims("./figs/borg-benign.jsonl", simulators, done)
}
