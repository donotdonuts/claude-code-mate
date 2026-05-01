package account

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Account struct {
	Tier          string // raw, e.g. "default_claude_max_5x"
	PlanName      string // "Pro" | "Max5" | "Max20" | tier
	HasExtraUsage bool
	Email         string
}

func path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude.json")
}

func Load() (*Account, error) {
	data, err := os.ReadFile(path())
	if err != nil {
		return nil, err
	}
	var raw struct {
		OAuthAccount struct {
			OrganizationRateLimitTier string `json:"organizationRateLimitTier"`
			HasExtraUsageEnabled      bool   `json:"hasExtraUsageEnabled"`
			EmailAddress              string `json:"emailAddress"`
		} `json:"oauthAccount"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	a := &Account{
		Tier:          raw.OAuthAccount.OrganizationRateLimitTier,
		HasExtraUsage: raw.OAuthAccount.HasExtraUsageEnabled,
		Email:         raw.OAuthAccount.EmailAddress,
	}
	a.PlanName = planNameFromTier(a.Tier)
	return a, nil
}

func planNameFromTier(tier string) string {
	switch tier {
	case "default_claude_pro":
		return "Pro"
	case "default_claude_max_5x":
		return "Max5"
	case "default_claude_max_20x":
		return "Max20"
	case "":
		return ""
	default:
		return tier
	}
}
