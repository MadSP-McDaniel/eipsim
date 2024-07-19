package policies

import (
	"math"

	"github.com/MadSP-McDaniel/eipsim/types"
)

type segmentedPoolEntry struct {
	ip    types.IPAddress
	owner types.TenantId
	valid bool
	added types.Duration
}

type segmentedPoolTenantMeta struct {
	allocations uint64
	billedTime  types.Duration
	ownerPool   []*segmentedPoolEntry
}

const segmentedCooldownTime = 30 * types.Minute

/*
	SegmentedPool aims to heuristically separate tenants with long-running and short-running workloads.

It does this by tracking the average time that tenants keep IPs allocated for. Each time an IP is allocated, the tenant's IP count increases. When IPs are released, the tenant then banks however much time they kept the IP for. Divide these.

IPs also keep track of how long they've recently been used for. Each IP has a timer that counts down, and when the IP is released its timer is reset to *at minimum* the time it was held for.

The timer counts down slower than realtime, by a factor of TimerMultiplier.

When allocating IP addresses, tenants will preference IPs that have a current timer value similar to that of their average IP holding time.
*/
type SegmentedPool struct {
	allIPs        map[types.IPAddress]*segmentedPoolEntry
	ipTimers      map[types.IPAddress]types.Duration
	ownerPools    map[types.TenantId]*segmentedPoolTenantMeta
	cooldownQueue []*segmentedPoolEntry

	TimerMultiplier float64
	NegativeTimers  bool // Allow timers to go negative

	BasePolicy
}

func NewSegmentedPool(TimerMultiplier float64, allowNegative bool) types.PoolPolicy {
	p := &SegmentedPool{TimerMultiplier: TimerMultiplier, BasePolicy: BasePolicy{Type: "segmented"}, NegativeTimers: allowNegative}
	if allowNegative {
		p.BasePolicy.Type = "segmented-neg"
	}
	return p
}

func (t *SegmentedPool) Init(s types.Simulator) {
	t.ownerPools = map[types.TenantId]*segmentedPoolTenantMeta{}
	t.allIPs = map[types.IPAddress]*segmentedPoolEntry{}
	t.ipTimers = map[types.IPAddress]types.Duration{}
	if t.TimerMultiplier == 0 {
		t.TimerMultiplier = 1
	}
}

func (t *SegmentedPool) Seed(s types.Simulator, ip types.IPAddress) {
	entry := &segmentedPoolEntry{ip, types.NilTenant, true, math.MinInt64}
	t.allIPs[ip] = entry
}

func (t *SegmentedPool) getMeta(id types.TenantId) *segmentedPoolTenantMeta {
	tenantMeta, ok := t.ownerPools[id]
	if !ok {
		tenantMeta = &segmentedPoolTenantMeta{}
		t.ownerPools[id] = tenantMeta
	}
	return tenantMeta
}

func (t *SegmentedPool) GetIPTimer(s types.Simulator, ip types.IPAddress) types.Duration {
	timer := types.Duration(float64(t.ipTimers[ip]-s.GetTime()) / t.TimerMultiplier)
	return timer
}

func (t *SegmentedPool) GetTenantIPTimer(s types.Simulator, id types.TenantId) types.Duration {
	tenantMeta := t.getMeta(id)
	return types.Duration(float64(tenantMeta.billedTime) / float64(tenantMeta.allocations))
}

func (t *SegmentedPool) GetIP(s types.Simulator, tenantID types.TenantId) (ip types.IPAddress) {
	now := s.GetTime()
	// Moe timers out of the cooldown queue if they're old enough
	for len(t.cooldownQueue) > 0 && t.cooldownQueue[0].added+(segmentedCooldownTime) <= now {
		if t.cooldownQueue[0].valid {
			t.allIPs[t.cooldownQueue[0].ip] = t.cooldownQueue[0]
		}
		t.cooldownQueue = t.cooldownQueue[1:]
	}
	tenantMeta := t.getMeta(tenantID)
	tenantMeta.allocations++
	// This tenant has IPs tagged to them, we can take one of those
	for len(tenantMeta.ownerPool) > 0 {
		entry := tenantMeta.ownerPool[0]
		if entry.added+(segmentedCooldownTime) > s.GetTime() {
			// IPs are all too new, take from the main pool
			break
		}
		tenantMeta.ownerPool = tenantMeta.ownerPool[1:]
		if entry.valid {
			entry.valid = false
			delete(t.allIPs, entry.ip)
			return entry.ip
		}
	}
	// Take the oldest IP from someone else.
	var bestIP *segmentedPoolEntry
	// targetIPTimer is the time value of the IP timer which will lead to its duration being closest to tenant's billable time
	var targetIPTimer = s.GetTime() + types.Duration(float64(tenantMeta.billedTime)/float64(tenantMeta.allocations)*t.TimerMultiplier)
	tries := 0
	for _, meta := range t.allIPs {
		if !meta.valid {
			panic("Invalid meta in segmented ip pool")
		}
		tries++
		if tries > 50 {
			break
		}

		// Don't let timers go negative.
		if !t.NegativeTimers && t.ipTimers[meta.ip] < now {
			t.ipTimers[meta.ip] = now
		}

		// We want the IP timer to be as close as possible (TODO: without going over)
		if bestIP == nil {
			bestIP = meta
			continue
		} else if (t.ipTimers[meta.ip] - targetIPTimer).Abs() < (t.ipTimers[bestIP.ip] - targetIPTimer).Abs() {
			// As close as possible
			bestIP = meta
		}

	}
	if bestIP == nil {
		panic("No IPs available")
	}

	bestIP.valid = false
	delete(t.allIPs, bestIP.ip)
	return bestIP.ip
}

func (t *SegmentedPool) ReleaseIP(s types.Simulator, ip types.IPAddress, tenantID types.TenantId) {
	tenantMeta := t.getMeta(tenantID)
	info := s.GetInfo(ip)
	ownedDuration := s.GetTime() - info.AllocatedAt
	entry := &segmentedPoolEntry{
		ip,
		tenantID,
		true,
		s.GetTime(),
	}
	timer := s.GetTime() + types.Duration(float64(ownedDuration)*t.TimerMultiplier)
	if timer > t.ipTimers[ip] {
		t.ipTimers[ip] = timer
	}
	tenantMeta.billedTime += ownedDuration
	tenantMeta.ownerPool = append(tenantMeta.ownerPool, entry)
	t.cooldownQueue = append(t.cooldownQueue, entry)
}
