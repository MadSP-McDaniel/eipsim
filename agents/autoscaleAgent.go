package agents

import (
	"math"

	"github.com/MadSP-McDaniel/eipsim/types"
	"github.com/MadSP-McDaniel/eipsim/util"
)

const dailyTerms = 24 * 2

type autoscaleConfig struct {
	id      types.TenantId
	nMax    float64
	nMin    float64
	targets [dailyTerms]float64
	expires types.Duration
	//f       util.Fourier
	ips map[types.IPAddress]struct{}
}

type AutoscaleAgent struct {
	tenantAutoscales map[types.Duration][]*autoscaleConfig

	NumTenants  int
	MaxWait     int // Max timesteps between processing
	NMax        int
	NMin        int
	maxHeadroom float64
	TenantChurn types.Duration // Mean time between tenant churn

	maxTenantId int

	BaseAgent
}

func (a *AutoscaleAgent) getNewConfig(s types.Simulator) *autoscaleConfig {
	id := a.minID + types.TenantId(a.maxTenantId)
	a.maxTenantId++
	scale := math.Log(float64(a.NMax) / float64(a.NMin))
	nMax := int(float64(a.NMin) * math.Pow(math.E, s.Rand().Float64()*scale))
	nMin := s.Rand().Intn(nMax)
	f := util.RandomFourier(s.Rand(), 24)
	config := &autoscaleConfig{id: id, nMax: float64(nMax), nMin: float64(nMin), ips: map[types.IPAddress]struct{}{}}
	for i := 0; i < dailyTerms; i++ {
		fTime := float64(i) / float64(dailyTerms)
		config.targets[i] = f.Compute(fTime)
		// config.targets[i] = config.nMin + int(float64(config.nMax-config.nMin)*f.Compute(fTime))
	}
	if a.TenantChurn > 0 {
		config.expires = s.GetTime() + types.Duration(s.Rand().ExpFloat64()*float64(a.TenantChurn))
	} else {
		config.expires = math.MaxInt64
	}
	return config
}

func (a *AutoscaleAgent) Init(s types.Simulator, minID types.TenantId, maxID types.TenantId) {
	a.BaseAgent.Init(s, minID, maxID)
	a.tenantAutoscales = make(map[types.Duration][]*autoscaleConfig)
	t := s.GetTime()
	for i := 0; i < a.NumTenants; i++ {
		a.tenantAutoscales[t] = append(a.tenantAutoscales[t], a.getNewConfig(s))
	}
}

func (a *AutoscaleAgent) Process(s types.Simulator) {
	t := s.GetTime()
	// Current time of day in range [0,1]

	// Pull the tenants we should process this cycle
	toProcess := a.tenantAutoscales[t]
	delete(a.tenantAutoscales, t)

	//currentTimestep := (t * dailyTerms / types.Day) * types.Day / dailyTerms
	nextTimestep := (t*dailyTerms/types.Day + 1) * types.Day / dailyTerms

	for i := 0; i < len(toProcess); i++ {
		config := toProcess[i]
		if config.expires < t {
			// Release all IPs from this tenant as they are churning
			for ip := range config.ips {
				delete(config.ips, ip)
				s.ReleaseIP(ip, config.id, true)
			}
			// Generate a new config
			toProcess = append(toProcess, a.getNewConfig(s))
			continue
		}

		targetIndex := int(t * dailyTerms / types.Day)
		// nMax := min(max(config.nMax+s.Rand().Float64()-0.5, 0), config.nMax)
		// nMaxDiff := nMax - config.nMax
		// nMaxDiff = min(nMaxDiff, a.maxHeadroom)
		// a.maxHeadroom -= nMaxDiff
		// config.nMax += nMaxDiff
		// config.nMin = min(max(config.nMin+s.Rand().Float64()-0.5, 0), float64(a.NMax))

		targetIPs := int(config.nMin + float64(config.nMax-config.nMin)*config.targets[targetIndex%dailyTerms])

		// Allocate IPs as needed
		for len(config.ips) < targetIPs {
			config.ips[s.GetIP(config.id)] = struct{}{}
		}
		// Free IPs as needed
		for len(config.ips) > targetIPs {
			for ip := range config.ips {
				delete(config.ips, ip)
				s.ReleaseIP(ip, config.id, true)
				break
			}
		}
		//targetIndexDelta := 1
		//for ; config.targets[(targetIndex+targetIndexDelta)%dailyTerms] == targetIPs && targetIndexDelta < 24; targetIndexDelta++ {
		//}
		nextProcess := nextTimestep + types.Duration(s.Rand().Intn(int(types.Day)/dailyTerms))
		a.tenantAutoscales[nextProcess] = append(a.tenantAutoscales[nextProcess], config)
	}
}
