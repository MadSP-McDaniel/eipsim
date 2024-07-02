package agents

import (
	"math"
	"math/rand"
	"sort"

	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/policies"
	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"
)

type adversaryIpMeta struct {
	createdAt types.Duration
	tenantId  types.TenantId
	ip        types.IPAddress

	// Stats
	prevTenantCount int
	timeSinceReuse  int
	newIP           bool
	hasLatentConf   bool
	segmentTimer    types.Duration
}

type AdversarialAgent struct {
	allAllocs             []adversaryIpMeta
	oldestActiveAlloc     int
	statsIndex            int
	benignAllocs          int
	benignExploitedAllocs int

	uniques map[types.IPAddress]struct{}

	SegmentedPool *policies.SegmentedPool `json:"-"`

	// How many IPs to create in total throughout the simulation
	MaxCreated uint64
	// Max simulatenous IPs
	MaxIPs int
	// How long to hold each IP for
	HoldDuration types.Duration
	// Max IPs to create each cycle
	MaxPerCycle int
	// Number of allocations before moving to next tenant
	AllocationsPerTenant int
	// Number of tenants before looping back to first tenant
	MaxTenants int
	// Don't do any processing before this time
	StartTime types.Duration
	BaseAgent
}

/*
func NewAdversarialAgent(maxIPs int, maxCreated uint64, holdDuration types.Duration, maxPerCycle int, startTime types.Duration) *AdversarialAgent {
	return &AdversarialAgent{
		map[types.IPAddress]adversaryIpMeta{},
		map[types.IPAddress]struct{}{},
		nil,
		nil,
		0,
		maxCreated,
		maxIPs,
		holdDuration,
		maxPerCycle,
		0,
		startTime,
		BaseAgent{Type: "adversarial"},
	}
}*/

func (a *AdversarialAgent) GetID() string {
	return "adversary"
}

func (a *AdversarialAgent) Process(s types.Simulator) {
	r := s.Rand()
	t := s.GetTime()

	if t < a.StartTime {
		return
	}
	// Allocate up to maxPerCycle IP addresses
	for i := 0; len(a.allAllocs)-a.oldestActiveAlloc < a.MaxIPs && i < r.Int()%a.MaxPerCycle; i++ {
		if uint64(len(a.allAllocs)) >= a.MaxCreated && a.MaxCreated != 0 {
			s.Done()
			break
		}
		var meta adversaryIpMeta
		meta.createdAt = t
		meta.tenantId = a.minID + (types.TenantId(len(a.allAllocs)) / types.TenantId(a.AllocationsPerTenant) % types.TenantId(a.MaxTenants))

		meta.ip = s.GetIP(meta.tenantId)
		if a.SegmentedPool != nil {
			meta.segmentTimer = a.SegmentedPool.GetIPTimer(s, meta.ip)
		}
		_, existingIP := a.uniques[meta.ip]
		meta.newIP = !existingIP
		a.uniques[meta.ip] = struct{}{}

		info := s.GetInfo(meta.ip)

		meta.prevTenantCount = info.UniqueOwners()
		meta.timeSinceReuse = int(info.ReleasedBenign)
		meta.hasLatentConf = info.HasConfig(t, a.minID)
		a.allAllocs = append(a.allAllocs, meta)
	}
	// Free any IP addresses that have reached holdDuration age
	for a.oldestActiveAlloc < len(a.allAllocs) {
		meta := &a.allAllocs[a.oldestActiveAlloc]
		// Check if the oldest allocation is ready to be freed.
		if t > meta.createdAt+a.HoldDuration {
			s.ReleaseIP(meta.ip, meta.tenantId, false)
			a.oldestActiveAlloc++
		} else {
			break
		}
	}
}

func (a *AdversarialAgent) Init(s types.Simulator, minID types.TenantId, maxID types.TenantId) {
	a.BaseAgent.Init(s, minID, maxID)
	if a.AllocationsPerTenant <= 0 {
		a.AllocationsPerTenant = math.MaxInt
	}
	if a.MaxTenants <= 0 {
		a.MaxTenants = math.MaxInt
	}

	s.RegisterStatCollector(a.CollectStats)
	s.RegisterIPAllocationCallback(a.IPAllocationCallback)
	a.uniques = make(map[types.IPAddress]struct{})
}

func (a *AdversarialAgent) IPAllocationCallback(s types.Simulator, ip types.IPAddress, tenant types.TenantId) {
	if tenant >= a.minID && tenant < a.maxID {
		return
	}
	if s.GetTime() < a.StartTime {
		return
	}
	_, wasAdversarial := a.uniques[ip]
	a.benignAllocs++
	if wasAdversarial {
		a.benignExploitedAllocs++
	}
}

func (a *AdversarialAgent) CollectStats(s types.Simulator, stats map[string]interface{}) {
	if a.statsIndex == len(a.allAllocs) {
		stats["adversary"] = nil
		return
	}
	advStats := make(map[string]interface{})
	stats["adversary"] = advStats
	advStats["created"] = len(a.allAllocs) - a.statsIndex
	advStats["totalCreated"] = len(a.allAllocs)
	var sumCount uint64
	var sumSeconds uint64
	var numNewIps uint64
	var numNewLCs uint64
	numEntries := len(a.allAllocs) - a.statsIndex
	for i := a.statsIndex; i < len(a.allAllocs); i++ {
		meta := &a.allAllocs[i]
		sumSeconds += uint64(meta.timeSinceReuse)
		sumCount += uint64(meta.prevTenantCount)
		if meta.newIP {
			numNewIps++
			if meta.hasLatentConf {
				numNewLCs++
			}
		}
	}
	advStats["avgTimeSinceReuse"] = sumSeconds / uint64(numEntries)
	advStats["avgPrevTenants"] = sumCount / uint64(numEntries)
	advStats["totalUniques"] = len(a.uniques)
	advStats["newUniques"] = numNewIps
	advStats["newLatentConfs"] = numNewLCs
	advStats["adversaryBenignAllocs"] = a.benignAllocs
	advStats["adversaryBenignExploitedAllocs"] = a.benignExploitedAllocs

	a.statsIndex = len(a.allAllocs)
}

func (a *AdversarialAgent) Cleanup(s types.Simulator) {
	stats := s.GetOverallStats()
	if a.statsIndex == 0 {
		return
	}
	a.statsIndex = 0
	a.CollectStats(s, stats)

	cdf := []int{}
	for i := 0; i < 1000; i++ {
		cdf = append(cdf, int(a.allAllocs[rand.Intn(len(a.allAllocs))].segmentTimer))
	}
	sort.Ints(cdf)
	stats["adversarySegmentCDF"] = cdf
}
