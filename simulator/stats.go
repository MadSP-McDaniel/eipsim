package simulator

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/MadSP-McDaniel/eipsim/types"
)

func (s *Simulator) CollectSimulatorPeriodicStats(newStats map[string]interface{}) {
	newStats["availableIPs"] = s.AvailableIPs()
	newStats["allocated"] = s.WindowAllocated
	newStats["latentConf"] = s.WindowConf

	s.WindowAllocated = 0
	s.WindowConf = 0
	//log.Println(s.GetTime(), newStats)
}

func (s *Simulator) CollectSimulatorOverallStats() {
	s.OverallStats["maxUsedIPs"] = s.MaxUsedIPs
	s.OverallStats["allocated"] = s.GetAllocated()
	s.OverallStats["latentConf"] = s.SimStats.TotalConf

	s.collectAllocationDurationCDF()
	s.collectFreeDurationCDF()
}

func (s *Simulator) outputAllocationTrace() {
	log.Println("Sorting allocs...")
	sort.Slice(s.allAllocations, func(i, j int) bool {
		return s.allAllocations[i].at < s.allAllocations[j].at
	})
	log.Println("Done.")
	f, err := os.Create("allocs.csv")
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(f, "time, duration, tenant")
	for _, alloc := range s.allAllocations {
		fmt.Fprintf(f, "%d, %d, %d\n", alloc.at, alloc.heldFor, alloc.id)
	}
	f.Close()
}

func (s *Simulator) collectAllocationDurationCDF() {

	sort.Slice(s.allAllocations, func(i, j int) bool {
		return s.allAllocations[i].heldFor < s.allAllocations[j].heldFor
	})
	durations := []types.Duration{}
	if len(s.allAllocations) != 0 {
		for i := 0.0; i < 1.0; i += 0.001 {
			durations = append(durations, s.allAllocations[int(float64(len(s.allAllocations))*i)].heldFor)
		}
	}
	s.OverallStats["allocationDurationCDF"] = durations
}

func (s *Simulator) collectFreeDurationCDF() {
	freeFors := []types.Duration{}

	for i := 0; i < len(s.allAllocations); i++ {
		if s.allAllocations[i].freeFor == 0 {
			continue
		}
		freeFors = append(freeFors, s.allAllocations[i].freeFor)
	}

	sort.Slice(freeFors, func(i, j int) bool {
		return freeFors[i] < freeFors[j]
	})
	if len(freeFors) == 0 {
		return // Never reallocated any IPs
	}
	durations := []types.Duration{}
	for i := 0.0; i < 1.0; i += 0.001 {
		durations = append(durations, freeFors[int(float64(len(freeFors))*i)])
	}
	s.OverallStats["freeDurationCDF"] = durations
}
