package utils

type OptionBool struct {
	Val      bool
	HasValue bool
}

func (p *OptionBool) Value() bool {
	return p.HasValue && p.Val
}
