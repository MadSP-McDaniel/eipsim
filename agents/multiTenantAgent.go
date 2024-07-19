package agents

import (
	"fmt"

	"github.com/MadSP-McDaniel/eipsim/types"
)

type MultiTenantAgent struct {
	activeIPs    map[types.IPAddress]types.TenantId
	currentIPs   int
	nextChange   types.Duration
	totalcreated int

	MaxIPs            int
	MinIPs            int
	MaxChangeInterval types.Duration
	ID                string
	MaxPerCycle       int
	NumTenants        int
	BaseAgent
}

func NewMultiTenantAgent(maxIPs int, minIPs int, maxChangeInterval types.Duration, id string, maxPerCycle int, numTenants int) types.Agent {
	return &MultiTenantAgent{MaxIPs: maxIPs, MinIPs: minIPs, MaxChangeInterval: maxChangeInterval, ID: id, MaxPerCycle: maxPerCycle, activeIPs: map[types.IPAddress]types.TenantId{}, NumTenants: numTenants, BaseAgent: BaseAgent{Type: "multi"}}
}

func (a *MultiTenantAgent) SetIPs(s types.Simulator) {
	r := s.Rand()
	// The workload scales randomly over time.
	if s.GetTime() > a.nextChange {
		a.currentIPs = (r.Int() % (a.MaxIPs - a.MinIPs)) + a.MinIPs
		a.nextChange = s.GetTime() + types.Duration(r.Int63()%int64(a.MaxChangeInterval))
	}
}

func (a *MultiTenantAgent) GetID() string {
	a.totalcreated++
	return fmt.Sprintf("%s.%d", a.ID, a.totalcreated)
}
func (a *MultiTenantAgent) Process(s types.Simulator) {
	r := s.Rand()
	a.SetIPs(s)
	// Allocate up to maxPerCycle IP addresses
	for i := 0; len(a.activeIPs) < a.currentIPs && i < a.MaxPerCycle; i++ {
		id := a.minID + types.TenantId(r.Int()%a.NumTenants)
		ip := s.GetIP(id)
		a.activeIPs[ip] = id
	}
	// Free IPs down to desired count
	for i := 0; len(a.activeIPs) > a.currentIPs && i < a.MaxPerCycle; i++ {
		for ip, id := range a.activeIPs {
			delete(a.activeIPs, ip)
			s.ReleaseIP(ip, id, true)
			break
		}
	}
}
