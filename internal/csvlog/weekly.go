package csvlog

import (
	"encoding/json"
	"time"
)

// Weekly reads sessions.csv and returns aggregated per-model token totals
// for sessions whose start timestamp is at or after `since`. Keys are the
// raw model identifiers as they were recorded (e.g. "claude-opus-4-7"),
// or short names ("opus") on rows written before the breakdown switched
// to full ids — callers should humanize on display.
//
// Returns nil on any read/parse error so the caller can degrade gracefully.
func Weekly(since time.Time) map[string]int {
	rows, err := readAll(Path())
	if err != nil || len(rows) < 2 {
		return nil
	}
	cols := map[string]int{}
	for i, h := range rows[0] {
		cols[h] = i
	}
	startIdx, ok1 := cols["start"]
	bdIdx, ok2 := cols["model_breakdown"]
	if !ok1 || !ok2 {
		return nil
	}
	out := map[string]int{}
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		if startIdx >= len(r) || bdIdx >= len(r) {
			continue
		}
		t, err := time.Parse(time.RFC3339, r[startIdx])
		if err != nil || t.Before(since) {
			continue
		}
		var bd map[string]int
		if err := json.Unmarshal([]byte(r[bdIdx]), &bd); err != nil {
			continue
		}
		for k, v := range bd {
			out[k] += v
		}
	}
	return out
}

// WeeklyExcluding is like Weekly but skips the row whose session_id
// matches `excludeID`. Use this so the live session can be folded in
// from the in-memory stats without double counting the (possibly stale)
// CSV row from a previous statusline tick.
func WeeklyExcluding(since time.Time, excludeID string) map[string]int {
	rows, err := readAll(Path())
	if err != nil || len(rows) < 2 {
		return nil
	}
	cols := map[string]int{}
	for i, h := range rows[0] {
		cols[h] = i
	}
	idIdx, ok0 := cols["session_id"]
	startIdx, ok1 := cols["start"]
	bdIdx, ok2 := cols["model_breakdown"]
	if !ok0 || !ok1 || !ok2 {
		return nil
	}
	out := map[string]int{}
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		if idIdx >= len(r) || startIdx >= len(r) || bdIdx >= len(r) {
			continue
		}
		if excludeID != "" && r[idIdx] == excludeID {
			continue
		}
		t, err := time.Parse(time.RFC3339, r[startIdx])
		if err != nil || t.Before(since) {
			continue
		}
		var bd map[string]int
		if err := json.Unmarshal([]byte(r[bdIdx]), &bd); err != nil {
			continue
		}
		for k, v := range bd {
			out[k] += v
		}
	}
	return out
}
