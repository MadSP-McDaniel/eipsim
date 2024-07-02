package types

import (
	"github.com/datadog/hyperloglog"
)

type IPInfo struct {
	Address        IPAddress
	Released       Duration
	ReleasedBenign Duration
	Owners         *hyperloglog.HyperLogLog
	Configurations map[TenantId]Duration
	Owner          TenantId
	AllocatedAt    Duration
}

func (i *IPInfo) HasConfig(t Duration, tenantId TenantId) bool {
	for other, expiration := range i.Configurations {
		if expiration <= t {
			delete(i.Configurations, other)
		}
		if other != tenantId {
			return true
		}
	}
	return false
}

func (i *IPInfo) UniqueOwners() int {
	return int(i.Owners.Count())
}
