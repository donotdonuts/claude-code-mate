package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"ccstats/internal/pricing"
	"ccstats/internal/transcript"
)

type WindowStats struct {
	Anchor           time.Time
	Reset            time.Time
	TotalTokens      int
	BillableTokens   int // input + output; matches plan-limit accounting (cache excluded)
	TotalInput       int
	TotalOutput      int
	TotalCacheRead   int
	TotalCacheCreate int
	TotalCost        float64
	Messages         int
	BurnRate         float64
	Expired          bool
}

const WindowDuration = 5 * time.Hour

type msgPoint struct {
	ts          time.Time
	tokens      int
	input       int
	output      int
	cacheRead   int
	cacheCreate int
	cost        float64
}

func ComputeWindow(now time.Time) (*WindowStats, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	pattern := filepath.Join(home, ".claude", "projects", "*", "*.jsonl")
	files, _ := filepath.Glob(pattern)

	cutoff := now.Add(-24 * time.Hour)
	var msgs []msgPoint
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		_ = transcript.IterateFile(f, func(e *transcript.Entry) error {
			if e.Type != "assistant" || e.IsSidechain {
				return nil
			}
			ts := e.Time()
			if ts.IsZero() || ts.Before(cutoff) {
				return nil
			}
			var msg transcript.Message
			if err := json.Unmarshal(e.Message, &msg); err != nil || msg.Usage == nil {
				return nil
			}
			u := msg.Usage
			tokens := u.InputTokens + u.OutputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
			cost := pricing.ComputeCost(msg.Model, u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens)
			msgs = append(msgs, msgPoint{
				ts:          ts,
				tokens:      tokens,
				input:       u.InputTokens,
				output:      u.OutputTokens,
				cacheRead:   u.CacheReadInputTokens,
				cacheCreate: u.CacheCreationInputTokens,
				cost:        cost,
			})
			return nil
		})
	}

	if len(msgs) == 0 {
		return &WindowStats{Anchor: now, Reset: now.Add(WindowDuration)}, nil
	}

	sort.Slice(msgs, func(i, j int) bool { return msgs[i].ts.Before(msgs[j].ts) })

	blockStart := msgs[0].ts
	for _, m := range msgs {
		if m.ts.Sub(blockStart) > WindowDuration {
			blockStart = m.ts
		}
	}
	blockEnd := blockStart.Add(WindowDuration)

	w := &WindowStats{Anchor: blockStart, Reset: blockEnd, Expired: now.After(blockEnd)}
	earliest := now
	for _, m := range msgs {
		if m.ts.Before(blockStart) || m.ts.After(blockEnd) {
			continue
		}
		w.TotalTokens += m.tokens
		w.BillableTokens += m.input + m.output
		w.TotalInput += m.input
		w.TotalOutput += m.output
		w.TotalCacheRead += m.cacheRead
		w.TotalCacheCreate += m.cacheCreate
		w.TotalCost += m.cost
		w.Messages++
		if m.ts.Before(earliest) {
			earliest = m.ts
		}
	}
	durationMin := now.Sub(earliest).Minutes()
	if durationMin < 1 {
		durationMin = 1
	}
	w.BurnRate = float64(w.BillableTokens) / durationMin
	return w, nil
}
