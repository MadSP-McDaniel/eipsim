package policies

import "gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"

type FIFOPool struct {
	ips []types.IPAddress
	BasePolicy
}

func (f *FIFOPool) Init(s types.Simulator) {
}

func NewFIFOPool() types.PoolPolicy {
	return &FIFOPool{nil, BasePolicy{Type: "fifo"}}
}

func (f *FIFOPool) GetIP(s types.Simulator, id types.TenantId) (ip types.IPAddress) {
	ip, f.ips = f.ips[0], f.ips[1:]
	return
}

func (f *FIFOPool) Seed(s types.Simulator, ip types.IPAddress) {
	f.ips = append(f.ips, ip)
}

func (f *FIFOPool) ReleaseIP(s types.Simulator, ip types.IPAddress, _ types.TenantId) {
	f.ips = append(f.ips, ip)
}
