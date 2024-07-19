package agents

import (
	"encoding/json"
	"errors"

	"github.com/MadSP-McDaniel/eipsim/types"
)

type AgentWrapper struct {
	Type        string
	types.Agent `json:"-"`
}

func (a *AgentWrapper) UnmarshalJSON(b []byte) error {
	type w AgentWrapper

	err := json.Unmarshal(b, (*w)(a))
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	switch a.Type {
	case "multi":
		a.Agent = &MultiTenantAgent{}
	case "adversary":
		a.Agent = &AdversarialAgent{}
	case "dynamic":
		a.Agent = &DynamicTenantAgent{}
	default:
		return errors.New("unknown agent type")
	}
	err = json.Unmarshal(b, a.Agent)
	if err != nil {
		return err
	}
	return nil
}

func (a *AgentWrapper) MarshalJSON() ([]byte, error) {

	return json.Marshal(a.Agent)
}

type BaseAgent struct {
	minID, maxID types.TenantId
	Type         string
}

func (b *BaseAgent) GetType() string {
	return b.Type
}

func (b *BaseAgent) Init(s types.Simulator, minID types.TenantId, maxID types.TenantId) {
	b.minID = minID
	b.maxID = maxID
}
