package simulator

import (
	"math"
	"math/rand"

	"github.com/MadSP-McDaniel/eipsim/agents"
	"github.com/MadSP-McDaniel/eipsim/policies"
	"github.com/MadSP-McDaniel/eipsim/types"
	"github.com/MadSP-McDaniel/eipsim/util"
	"github.com/datadog/hyperloglog"
)

type allocation struct {
	at      types.Duration
	id      types.TenantId
	heldFor types.Duration
	freeFor types.Duration
}

type SimStats struct {

	// Window allocations
	WindowAllocated int
	WindowConf      int

	// Overall allocations
	TotalConf int

	MaxUsedIPs int
}

type Simulator struct {
	ipMeta  map[types.IPAddress]*types.IPInfo
	freeIPs map[types.IPAddress]struct{}
	t       types.Duration
	done    bool

	// Statistics
	allocated     int
	released      int
	totalTimeHeld types.Duration

	allAllocations         []allocation
	AllocationSamplingRate int

	SimStats `json:"-"`

	// IPs are first seeded to tenants in order before using the pool, so that all IPs have some tenant associated with them.
	TimeDelta types.Duration
	MaxTime   types.Duration

	Agents []agents.AgentWrapper

	Policy policies.PoolPolicyWrapper

	LatentConfProbability float64

	TotalIPs int

	rand            *rand.Rand
	TimeSeriesStats map[types.Duration]map[string]interface{}
	OverallStats    map[string]interface{}

	StatCollectionInterval types.Duration

	statCollectors        []types.StatCollector
	ipAllocationCallbacks []types.IPAllocationCallback
}

func (s *Simulator) GetTimeDelta() types.Duration {
	return s.TimeDelta
}

func NewSimulator(ips int, p types.PoolPolicy, ts types.Duration) *Simulator {
	s := &Simulator{ipMeta: make(map[types.IPAddress]*types.IPInfo), freeIPs: make(map[types.IPAddress]struct{}), Policy: policies.PoolPolicyWrapper{Type: p.GetType(), PoolPolicy: p}, TotalIPs: ips, TimeDelta: ts}

	return s
}

func (s *Simulator) RegisterStatCollector(sc types.StatCollector) {
	s.statCollectors = append(s.statCollectors, sc)
}

func (s *Simulator) RegisterIPAllocationCallback(cb types.IPAllocationCallback) {
	s.ipAllocationCallbacks = append(s.ipAllocationCallbacks, cb)
}

func (s *Simulator) GetTime() types.Duration {
	return s.t
}

// Done instructs the pool to quit running after the next cycle
func (s *Simulator) Done() {
	s.done = true
}

func (s *Simulator) CollectPeriodicStats() {
	if s.StatCollectionInterval == 0 || s.t%s.StatCollectionInterval != 0 {
		return
	}
	if s.TimeSeriesStats == nil {
		s.TimeSeriesStats = make(map[types.Duration]map[string]interface{})
	}
	newStats := make(map[string]interface{})
	for _, sc := range s.statCollectors {
		sc(s, newStats)
	}

	s.CollectSimulatorPeriodicStats(newStats)
	s.TimeSeriesStats[s.t] = newStats
}

func (s *Simulator) Process() bool {
	if s.MaxTime != 0 && s.t >= s.MaxTime {
		return false
	}
	if s.done {
		return false
	}
	for _, agent := range s.Agents {
		agent.Process(s)
	}
	s.t = s.t + s.TimeDelta
	s.CollectPeriodicStats()
	return true
}

func (s *Simulator) Rand() *rand.Rand {
	return s.rand
}

func (s *Simulator) InitAgents() {
	s.rand = rand.New(rand.NewSource(0))
	s.Policy.Init(s)
	for len(s.ipMeta) < s.TotalIPs {
		ip := types.IPAddress(len(s.ipMeta))
		hll, err := hyperloglog.New(16)
		if err != nil {
			panic(err)
		}
		s.ipMeta[ip] = &types.IPInfo{Address: ip, Released: 0, Owners: hll, Configurations: make(map[types.TenantId]types.Duration), Owner: types.NilTenant}
		s.Policy.Seed(s, ip)
		s.freeIPs[ip] = struct{}{}
	}
	idAllocSize := types.TenantId(math.MaxUint32) / types.TenantId(len(s.Agents)) / 3
	id := types.TenantId(1)
	for _, agent := range s.Agents {
		agent.Init(s, id, id+idAllocSize)
		id += 2 * idAllocSize
	}
}

func (s *Simulator) CleanupAgents() {
	s.OverallStats = make(map[string]interface{})
	for _, agent := range s.Agents {
		if c, ok := agent.Agent.(types.Cleanuper); ok {
			c.Cleanup(s)
		}
	}
	s.CollectSimulatorOverallStats()
}

func (s *Simulator) ProcessAll() {
	s.InitAgents()
	for s.Process() {
	}
	s.CleanupAgents()
}

func (s *Simulator) GetAllocated() int {
	return s.allocated
}

func (s *Simulator) GetTotalTimeHeld() types.Duration {
	return s.totalTimeHeld
}

func (s *Simulator) GetReleased() int {
	return s.released
}

func (s *Simulator) GetOverallStats() map[string]interface{} {
	return s.OverallStats
}

func (s *Simulator) GetIP(tenantID types.TenantId) (ip types.IPAddress) {
	s.allocated++
	// No remaining seed IPs, pull from the pool
	ip = s.Policy.GetIP(s, tenantID)
	if _, ok := s.freeIPs[ip]; !ok {
		panic("Pool returned IP address that isn't free")
	}
	delete(s.freeIPs, ip)

	s.ipMeta[ip].Owner = tenantID
	s.ipMeta[ip].AllocatedAt = s.t
	s.WindowAllocated += 1
	if s.ipMeta[ip].HasConfig(s.t, tenantID) {
		s.WindowConf += 1
		s.TotalConf += 1
	}
	// Track Max Used IPs
	usedIps := s.TotalIPs - len(s.freeIPs)
	if usedIps > s.MaxUsedIPs {
		s.MaxUsedIPs = usedIps
	}
	return ip
}

func (s *Simulator) GetInfo(ip types.IPAddress) *types.IPInfo {
	return s.ipMeta[ip]
}

func (s *Simulator) GetAllMeta() map[types.IPAddress]*types.IPInfo {
	return s.ipMeta
}

func (s *Simulator) ReleaseIP(ip types.IPAddress, tenantID types.TenantId, benign bool) {
	s.released++
	if s.ipMeta[ip].Owner != tenantID {
		panic("Tenant returned IP that didn't belong to it")
	}
	if _, ok := s.freeIPs[ip]; ok {
		panic("Tenant released free IP")
	}
	freeFor := s.ipMeta[ip].AllocatedAt - s.ipMeta[ip].Released
	if s.ipMeta[ip].Released == 0 {
		freeFor = 0
	}
	s.ipMeta[ip].Released = s.GetTime()
	if benign { // Track last ownership and latent config for benign tenants
		s.ipMeta[ip].ReleasedBenign = s.GetTime()
		s.ipMeta[ip].Owners.Add(uint32(tenantID))
		if s.rand.Float64() < s.LatentConfProbability {
			expirationTime := s.GetTime() + types.Duration(util.SampleExponential(s.rand, 1/float64(s.GetTime()-s.ipMeta[ip].AllocatedAt)))
			s.ipMeta[ip].Configurations[tenantID] = expirationTime
		}
	}

	s.ipMeta[ip].Owner = types.NilTenant
	s.totalTimeHeld += s.GetTime() - s.ipMeta[ip].AllocatedAt
	s.Policy.ReleaseIP(s, ip, tenantID)
	s.freeIPs[ip] = struct{}{}
	for _, cb := range s.ipAllocationCallbacks {
		cb(s, ip, tenantID)
	}
	if s.AllocationSamplingRate == 0 || s.rand.Intn(s.AllocationSamplingRate) == 0 {
		s.allAllocations = append(s.allAllocations,
			allocation{
				s.ipMeta[ip].AllocatedAt,
				tenantID,
				s.GetTime() - s.ipMeta[ip].AllocatedAt,
				freeFor,
			})
	}
}

func (s *Simulator) AddAgent(agent types.Agent) {
	s.Agents = append(s.Agents, agents.AgentWrapper{Type: agent.GetType(), Agent: agent})
}

func (s *Simulator) AvailableIPs() uint32 {
	return uint32(len(s.freeIPs) + int(s.TotalIPs) - len(s.ipMeta))
}

// Run creates and executes a simulator
// func Run() {
// 	pools := []types.PoolPolicy{policies.NewTaggedPool() /*, NewFIFOPool(),*policies.NewTaggedPool()*/}
// 	for _, pool := range pools {
// 		s := NewSimulator(700000, pool, 1)
// 		s.MaxTime = 100 * types.Day
// 		s.LatentConfProbability = 0.1
// 		//s.AddAgent(agents.NewMultiTenantAgent(500000, 1000, 10*types.Minute, "test", 100, 100000))
// 		s.AddAgent(&agents.AutoscaleAgent{NumTenants: 120000, MaxWait: 3600, NMax: 30, NMin: 1, BaseAgent: agents.BaseAgent{Type: "autoscale"}, TenantChurn: 365 * types.Day})
// 		//adversary := agents.NewAdversarialAgent(60, 500000, 10*types.Minute, 10, 10*types.Day)
// 		//s.AddAgent(adversary)
// 		b, err := json.MarshalIndent(s, "", "\t")
// 		fmt.Println(string(b), err)
// 		s.ProcessAll()
// 		//fmt.Println(pool.GetType())
// 		//adversary.PrintStats()
// 	}

// }
