package pricing

import "strings"

// Prices in USD per 1M tokens. Approximates Anthropic's public pricing.
type ModelPrice struct {
	Input         float64
	Output        float64
	CacheRead     float64
	CacheCreate5m float64
	CacheCreate1h float64
	ContextWindow int
}

var table = map[string]ModelPrice{
	"opus":   {Input: 15, Output: 75, CacheRead: 1.50, CacheCreate5m: 18.75, CacheCreate1h: 30, ContextWindow: 200_000},
	"sonnet": {Input: 3, Output: 15, CacheRead: 0.30, CacheCreate5m: 3.75, CacheCreate1h: 6, ContextWindow: 200_000},
	"haiku":  {Input: 1, Output: 5, CacheRead: 0.10, CacheCreate5m: 1.25, CacheCreate1h: 2, ContextWindow: 200_000},
}

func For(model string) ModelPrice {
	m := strings.ToLower(model)
	var p ModelPrice
	switch {
	case strings.Contains(m, "opus"):
		p = table["opus"]
	case strings.Contains(m, "sonnet"):
		p = table["sonnet"]
	case strings.Contains(m, "haiku"):
		p = table["haiku"]
	default:
		p = table["sonnet"]
	}
	if strings.Contains(m, "1m") || strings.Contains(m, "[1m]") {
		p.ContextWindow = 1_000_000
	}
	return p
}

func ComputeCost(model string, input, output, cacheRead, cacheCreate int) float64 {
	p := For(model)
	return float64(input)*p.Input/1e6 +
		float64(output)*p.Output/1e6 +
		float64(cacheRead)*p.CacheRead/1e6 +
		float64(cacheCreate)*p.CacheCreate5m/1e6
}
