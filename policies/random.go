package policies

import (
	"github.com/MadSP-McDaniel/eipsim/types"
)

type randomPoolQueueEntry struct {
	types.IPAddress
	types.Duration
}

type RandomPool struct {
	ips   map[types.IPAddress]struct{}
	queue []randomPoolQueueEntry
	BasePolicy
	MinAvailable int
}

func NewRandomPool() types.PoolPolicy {
	return &RandomPool{BasePolicy: BasePolicy{Type: "random"}}
}

func (r *RandomPool) Init(s types.Simulator) {
	r.ips = map[types.IPAddress]struct{}{}
}

func (r *RandomPool) Seed(s types.Simulator, ip types.IPAddress) {
	r.ips[ip] = struct{}{}
	r.MinAvailable = len(r.ips)
}

func (r *RandomPool) GetIP(s types.Simulator, id types.TenantId) types.IPAddress {
	t := s.GetTime()
	for len(r.queue) > 0 && r.queue[0].Duration+(30*types.Minute) <= t {
		r.ips[r.queue[0].IPAddress] = struct{}{}
		r.queue = r.queue[1:]
	}
	if len(r.ips) < int(r.MinAvailable) {
		r.MinAvailable = len(r.ips)
	}
	for k, _ := range r.ips {
		delete(r.ips, k)
		return k
	}
	panic("IP pool ran out of addresses")
}

func (r *RandomPool) ReleaseIP(s types.Simulator, ip types.IPAddress, _ types.TenantId) {
	r.queue = append(r.queue, randomPoolQueueEntry{ip, s.GetTime()})
}
