package types

import (
	"fmt"
	"math/rand"
	"time"
)

type IPAddress uint32

type Simulator interface {
	GetTime() Duration
	GetIP(TenantId) IPAddress
	ReleaseIP(IPAddress, TenantId, bool)
	GetInfo(IPAddress) *IPInfo
	Done()
	AvailableIPs() uint32
	GetAllocated() int
	GetTotalTimeHeld() Duration
	GetReleased() int
	Rand() *rand.Rand
	RegisterStatCollector(StatCollector)
	RegisterIPAllocationCallback(IPAllocationCallback)
	GetTimeDelta() Duration
	GetOverallStats() map[string]interface{}
}

type PoolPolicy interface {
	// GetIP requests an IP for a given tenant
	GetIP(Simulator, TenantId) IPAddress
	// ReleaseIP releases an IP used by a given tenant
	ReleaseIP(Simulator, IPAddress, TenantId)
	GetType() string
	Init(Simulator)
	Seed(Simulator, IPAddress)
}

type Duration int64

const Second Duration = 1
const Minute Duration = 60 * Second
const Hour Duration = 60 * Minute
const Day Duration = 24 * Hour

func (x Duration) Abs() Duration {
	if x < 0 {
		return -x
	}
	return x
}

type StatCollector func(Simulator, map[string]interface{})

type IPAllocationCallback func(Simulator, IPAddress, TenantId)

type Agent interface {
	Process(Simulator)
	GetType() string
	Init(Simulator, TenantId, TenantId)
}

type Cleanuper interface {
	Cleanup(Simulator)
}

type TenantId uint32

var NilTenant TenantId = 0

func (d Duration) String() string {
	if d >= Day {
		return fmt.Sprintf("%dd%s", d/Day, (time.Duration(d%Day) * time.Second))
	}
	return (time.Duration(d) * time.Second).String()
}
