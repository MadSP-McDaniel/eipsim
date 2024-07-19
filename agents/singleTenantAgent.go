package agents

import (
	"github.com/MadSP-McDaniel/eipsim/types"
)

type DynamicTenantAgent struct {
	activeIPs         map[types.IPAddress]struct{}
	maxIPs            int
	minIPs            int
	currentIPs        int
	nextChange        types.Duration
	maxChangeInterval types.Duration
	id                string
	maxPerCycle       int
	BaseAgent
}

func (a *DynamicTenantAgent) SetIPs(s types.Simulator) {
	// The workload scales randomly over time.
	if s.GetTime() > a.nextChange {
		a.currentIPs = (s.Rand().Int() % (a.maxIPs - a.minIPs)) + a.minIPs
		a.nextChange = s.GetTime() + types.Duration(s.Rand().Int63())%a.maxChangeInterval
	}
}

func (a *DynamicTenantAgent) Type() string {
	return "dynamic"
}

func (a *DynamicTenantAgent) GetID() string {
	return a.id
}

func (a *DynamicTenantAgent) Process(s types.Simulator) {
	a.SetIPs(s)
	// Allocate up to maxPerCycle IP addresses
	for i := 0; len(a.activeIPs) < a.currentIPs && i < a.maxPerCycle; i++ {
		ip := s.GetIP(a.minID)
		a.activeIPs[ip] = struct{}{}
	}
	// Free any IP addresses that have reached holdDuration age
	for i := 0; len(a.activeIPs) > a.currentIPs && i < a.maxPerCycle; i++ {
		for ip := range a.activeIPs {
			delete(a.activeIPs, ip)
			s.ReleaseIP(ip, a.minID, true)
		}
	}
}
