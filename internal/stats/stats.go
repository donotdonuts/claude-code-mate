package stats

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"ccstats/internal/pricing"
	"ccstats/internal/transcript"
)

type ModelUsage struct {
	Model               string
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	Cost                float64
	Turns               int
}

type SessionStats struct {
	SessionID        string
	Model            string
	Start            time.Time
	LastActivity     time.Time
	Turns            int
	Words            int
	LOCDelta         int
	TotalInput       int
	TotalOutput      int
	TotalCacheRead   int
	TotalCacheCreate int
	TotalCost        float64
	LastCtxTokens    int
	Models           map[string]*ModelUsage
	BurnRate         float64
	ToolFailures     int
}

type ModelShare struct {
	Model string
	Pct   float64
}

func (s *SessionStats) Duration() time.Duration {
	if s.Start.IsZero() || s.LastActivity.IsZero() {
		return 0
	}
	return s.LastActivity.Sub(s.Start)
}

func (s *SessionStats) DurationMin() float64 {
	return s.Duration().Minutes()
}

// Finalize stamps the session's "as of" time to now and recomputes the burn
// rate as total session tokens divided by wall-clock minutes since the first
// transcript entry. Call this once after Compute so the statusline shows
// elapsed time since the user opened Claude in the terminal — not just the
// span between the first and last assistant message — and so the burn rate
// stays consistent with that elapsed time even when sessions span multiple
// 5-hour rate-limit windows.
func (s *SessionStats) Finalize(now time.Time) {
	if s.Start.IsZero() {
		return
	}
	s.LastActivity = now
	mins := s.DurationMin()
	if mins < 1 {
		mins = 1
	}
	s.BurnRate = float64(s.TotalTokens()) / mins
}

func (s *SessionStats) TotalTokens() int {
	return s.TotalInput + s.TotalOutput + s.TotalCacheRead + s.TotalCacheCreate
}

func (s *SessionStats) ContextPct() float64 {
	if s.LastCtxTokens == 0 {
		return 0
	}
	cw := pricing.For(s.Model).ContextWindow
	if cw == 0 {
		return 0
	}
	return float64(s.LastCtxTokens) / float64(cw) * 100
}

func (s *SessionStats) ContextWindow() int {
	return pricing.For(s.Model).ContextWindow
}

func (s *SessionStats) ModelBreakdown() []ModelShare {
	total := 0
	for _, m := range s.Models {
		total += m.InputTokens + m.OutputTokens + m.CacheReadTokens + m.CacheCreationTokens
	}
	if total == 0 {
		return nil
	}
	out := make([]ModelShare, 0, len(s.Models))
	for _, m := range s.Models {
		used := m.InputTokens + m.OutputTokens + m.CacheReadTokens + m.CacheCreationTokens
		out = append(out, ModelShare{Model: ShortName(m.Model), Pct: float64(used) / float64(total) * 100})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pct > out[j].Pct })
	return out
}

func ShortName(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"):
		return "opus"
	case strings.Contains(m, "sonnet"):
		return "sonnet"
	case strings.Contains(m, "haiku"):
		return "haiku"
	}
	return model
}

func Compute(transcriptPath string) (*SessionStats, error) {
	s := &SessionStats{Models: make(map[string]*ModelUsage)}

	err := transcript.IterateFile(transcriptPath, func(e *transcript.Entry) error {
		if e.IsSidechain {
			return nil
		}
		ts := e.Time()
		if !ts.IsZero() {
			if s.Start.IsZero() {
				s.Start = ts
			}
			s.LastActivity = ts
		}
		if e.SessionID != "" {
			s.SessionID = e.SessionID
		}

		switch e.Type {
		case "user":
			var msg struct {
				Content json.RawMessage `json:"content"`
				Role    string          `json:"role"`
			}
			if err := json.Unmarshal(e.Message, &msg); err == nil {
				if msg.Role == "user" || msg.Role == "" {
					s.Words += countWordsInUserContent(msg.Content)
					s.ToolFailures += countToolFailuresInContent(msg.Content)
				}
			}
		case "assistant":
			var msg transcript.Message
			if err := json.Unmarshal(e.Message, &msg); err != nil {
				return nil
			}
			s.Turns++
			if msg.Model != "" {
				s.Model = msg.Model
			}
			if msg.Usage != nil {
				u := *msg.Usage
				s.TotalInput += u.InputTokens
				s.TotalOutput += u.OutputTokens
				s.TotalCacheRead += u.CacheReadInputTokens
				s.TotalCacheCreate += u.CacheCreationInputTokens
				cost := pricing.ComputeCost(msg.Model, u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens)
				s.TotalCost += cost
				s.LastCtxTokens = u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens

				key := msg.Model
				m, ok := s.Models[key]
				if !ok {
					m = &ModelUsage{Model: key}
					s.Models[key] = m
				}
				m.InputTokens += u.InputTokens
				m.OutputTokens += u.OutputTokens
				m.CacheReadTokens += u.CacheReadInputTokens
				m.CacheCreationTokens += u.CacheCreationInputTokens
				m.Cost += cost
				m.Turns++
			}
			s.LOCDelta += countLOCInAssistantContent(msg.Content)
		}
		return nil
	})

	return s, err
}

func countWordsInUserContent(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return len(strings.Fields(s))
	}
	var blocks []transcript.ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return 0
	}
	n := 0
	for _, b := range blocks {
		if b.Type == "text" {
			n += len(strings.Fields(b.Text))
		}
	}
	return n
}

func countToolFailuresInContent(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var blocks []transcript.ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return 0
	}
	n := 0
	for _, b := range blocks {
		if b.Type == "tool_result" && b.IsError {
			n++
		}
	}
	return n
}

func countLOCInAssistantContent(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var blocks []transcript.ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return 0
	}
	delta := 0
	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}
		switch b.Name {
		case "Write":
			var input struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(b.Input, &input); err == nil {
				delta += linesIn(input.Content)
			}
		case "Edit":
			var input struct {
				OldString string `json:"old_string"`
				NewString string `json:"new_string"`
			}
			if err := json.Unmarshal(b.Input, &input); err == nil {
				delta += linesIn(input.NewString) - linesIn(input.OldString)
			}
		case "NotebookEdit":
			var input struct {
				NewSource string `json:"new_source"`
				OldSource string `json:"old_source"`
			}
			if err := json.Unmarshal(b.Input, &input); err == nil {
				delta += linesIn(input.NewSource) - linesIn(input.OldSource)
			}
		}
	}
	return delta
}

func linesIn(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
