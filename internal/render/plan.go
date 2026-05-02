package render

import "ccmate/internal/account"

type Plan struct {
	Name       string
	TokenLimit int
	CostLimit  float64
}

var (
	Pro   = Plan{Name: "Pro", TokenLimit: 19_000, CostLimit: 18}
	Max5  = Plan{Name: "Max5", TokenLimit: 88_000, CostLimit: 35}
	Max20 = Plan{Name: "Max20", TokenLimit: 220_000, CostLimit: 140}
)

// PlanFor returns the best-known limits for a plan name (used by detail TUI's
// estimated-cost section). The statusline does not consult these any more —
// it reads `rate_limits.five_hour.used_percentage` directly from the payload.
func PlanFor(name string) Plan {
	switch name {
	case "Pro":
		return Pro
	case "Max5":
		return Max5
	case "Max20":
		return Max20
	}
	return Plan{Name: name}
}

func PlanFromAccount(a *account.Account) Plan {
	if a == nil {
		return Pro
	}
	return PlanFor(a.PlanName)
}
