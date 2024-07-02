package policies

import (
	"encoding/json"
	"errors"

	"gitlab-siis.cse.psu.edu/cloud-squatting/ipsim/types"
)

type PoolPolicyWrapper struct {
	Type             string
	types.PoolPolicy `json:"-"`
}

func (p *PoolPolicyWrapper) UnmarshalJSON(b []byte) error {
	type w PoolPolicyWrapper

	err := json.Unmarshal(b, (*w)(p))
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}
	switch p.Type {
	case "random":
		p.PoolPolicy = &RandomPool{}
	case "fifo":
		p.PoolPolicy = &FIFOPool{}
	case "tagged":
		p.PoolPolicy = &TaggedPool{}
	case "segmented":
		p.PoolPolicy = &SegmentedPool{}
	default:
		return errors.New("unknown agent type")
	}
	err = json.Unmarshal(b, p.PoolPolicy)
	if err != nil {
		return err
	}
	return nil
}

func (p *PoolPolicyWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.PoolPolicy)
}

type BasePolicy struct {
	Type string
}

func (b *BasePolicy) GetType() string {
	return b.Type
}
