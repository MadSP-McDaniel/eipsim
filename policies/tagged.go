package policies

import "gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"

type taggedPoolEntry struct {
	ip    types.IPAddress
	owner types.TenantId
	valid bool
	added types.Duration
}

// TaggedPool uses an arraylist to track a FIFO queue of IP ownership. It additionally holds FIFO queues for each tenant in slices. Tenant IPs are initially drawn from their slices, then from the pool at large.
type TaggedPool struct {
	allIPs     []*taggedPoolEntry
	ownerPools map[types.TenantId][]*taggedPoolEntry
	BasePolicy
}

func NewTaggedPool() types.PoolPolicy {
	return &TaggedPool{BasePolicy: BasePolicy{Type: "tagged"}}
}

func (t *TaggedPool) Init(s types.Simulator) {
	t.ownerPools = map[types.TenantId][]*taggedPoolEntry{}
}

func (t *TaggedPool) Seed(s types.Simulator, ip types.IPAddress) {
	entry := &taggedPoolEntry{ip, types.NilTenant, true, s.GetTime()}
	t.allIPs = append(t.allIPs, entry)
}

func (t *TaggedPool) GetIP(s types.Simulator, tenantID types.TenantId) (ip types.IPAddress) {
	// This tenant has IPs tagged to them, we can take one of those
	for len(t.ownerPools[tenantID]) > 0 {
		entry := t.ownerPools[tenantID][0]
		if entry.added+(30*types.Minute) > s.GetTime() {
			// IPs are all too new, take from the main pool
			break
		}
		t.ownerPools[tenantID] = t.ownerPools[tenantID][1:]
		if entry.valid {
			entry.valid = false
			return entry.ip
		}
	}
	// Take the oldest IP from someone else.
	for len(t.allIPs) > 0 {
		entry := t.allIPs[0]
		t.allIPs = t.allIPs[1:]
		if entry.valid {
			entry.valid = false
			return entry.ip
		}
	}
	panic("No IPs available")
}

func (t *TaggedPool) ReleaseIP(s types.Simulator, ip types.IPAddress, tenantID types.TenantId) {
	entry := &taggedPoolEntry{ip, tenantID, true, s.GetTime()}
	t.ownerPools[tenantID] = append(t.ownerPools[tenantID], entry)
	t.allIPs = append(t.allIPs, entry)
}
