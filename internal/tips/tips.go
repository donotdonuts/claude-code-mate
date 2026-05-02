package tips

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Tip struct {
	ID       string
	Trigger  string
	Priority int
	Cooldown int
	Message  string
}

type Vars struct {
	CtxPct             float64
	SessionMinutes     float64
	Turns              int
	TotalTokens        int
	TotalCost          float64
	OverageCost        float64
	BurnRate           float64
	ToolFailuresRecent int
	Model              string
}

func DefaultDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "ccmate", "tips")
}

func Load(dir string) ([]Tip, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Tip
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		t, err := loadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Priority > out[j].Priority })
	return out, nil
}

func loadFile(path string) (Tip, error) {
	t := Tip{
		ID:       strings.TrimSuffix(filepath.Base(path), ".md"),
		Priority: 50,
		Cooldown: 5,
	}
	f, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	state := 0 // 0=preamble, 1=in frontmatter, 2=body
	var fmLines, bodyLines []string
	for scanner.Scan() {
		line := scanner.Text()
		switch state {
		case 0:
			if strings.TrimSpace(line) == "---" {
				state = 1
			} else if strings.TrimSpace(line) != "" {
				bodyLines = append(bodyLines, line)
				state = 2
			}
		case 1:
			if strings.TrimSpace(line) == "---" {
				state = 2
				continue
			}
			fmLines = append(fmLines, line)
		case 2:
			bodyLines = append(bodyLines, line)
		}
	}
	for _, l := range fmLines {
		kv := strings.SplitN(l, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		val = strings.Trim(val, `"'`)
		switch key {
		case "id":
			t.ID = val
		case "trigger":
			t.Trigger = val
		case "priority":
			if n, err := strconv.Atoi(val); err == nil {
				t.Priority = n
			}
		case "cooldown_turns":
			if n, err := strconv.Atoi(val); err == nil {
				t.Cooldown = n
			}
		}
	}
	t.Message = strings.TrimSpace(strings.Join(bodyLines, " "))
	t.Message = collapseWhitespace(t.Message)
	return t, nil
}

func collapseWhitespace(s string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

func Pick(tips []Tip, v Vars, lastShown map[string]int, currentTurn int) *Tip {
	for i := range tips {
		t := &tips[i]
		if last, seen := lastShown[t.ID]; seen && currentTurn-last < t.Cooldown {
			continue
		}
		if matches(t.Trigger, v) {
			return t
		}
	}
	return nil
}

// PickSticky picks the tip to render with sticky behavior: once a tip appears,
// it stays on screen across subsequent turns. It is replaced only when a
// different tip's trigger fires (highest-priority wins among current matches).
// If no tip currently matches, the previously-shown tip is kept on screen.
func PickSticky(loaded []Tip, v Vars, state State) *Tip {
	var best *Tip
	for i := range loaded {
		t := &loaded[i]
		if matches(t.Trigger, v) {
			if best == nil || t.Priority > best.Priority {
				best = t
			}
		}
	}
	if best != nil {
		return best
	}
	if state.CurrentTipID == "" {
		return nil
	}
	for i := range loaded {
		if loaded[i].ID == state.CurrentTipID {
			return &loaded[i]
		}
	}
	return nil
}

func matches(expr string, v Vars) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}
	for _, p := range strings.Split(expr, "&&") {
		if !evalSimple(strings.TrimSpace(p), v) {
			return false
		}
	}
	return true
}

var cmpRe = regexp.MustCompile(`^([a-z_]+)\s*(>=|<=|==|!=|>|<)\s*(.+)$`)

func evalSimple(p string, v Vars) bool {
	m := cmpRe.FindStringSubmatch(p)
	if m == nil {
		return false
	}
	name, op, rhs := m[1], m[2], strings.TrimSpace(m[3])
	rhs = strings.Trim(rhs, `"'`)

	if name == "model" {
		lhs := strings.ToLower(v.Model)
		want := strings.ToLower(rhs)
		switch op {
		case "==":
			return strings.Contains(lhs, want)
		case "!=":
			return !strings.Contains(lhs, want)
		}
		return false
	}

	var lhs float64
	switch name {
	case "ctx_pct":
		lhs = v.CtxPct
	case "session_minutes":
		lhs = v.SessionMinutes
	case "turns":
		lhs = float64(v.Turns)
	case "total_tokens":
		lhs = float64(v.TotalTokens)
	case "total_cost":
		lhs = v.TotalCost
	case "overage_cost":
		lhs = v.OverageCost
	case "burn_rate":
		lhs = v.BurnRate
	case "tool_failures_recent":
		lhs = float64(v.ToolFailuresRecent)
	default:
		return false
	}
	rhsF, err := strconv.ParseFloat(rhs, 64)
	if err != nil {
		return false
	}
	switch op {
	case ">":
		return lhs > rhsF
	case ">=":
		return lhs >= rhsF
	case "<":
		return lhs < rhsF
	case "<=":
		return lhs <= rhsF
	case "==":
		return lhs == rhsF
	case "!=":
		return lhs != rhsF
	}
	return false
}

type State struct {
	LastShown    map[string]int `json:"last_shown"`
	LastTurn     int            `json:"last_turn"`
	CurrentTipID string         `json:"current_tip_id"`
}

func StatePath(sessionID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "ccmate", "state", sessionID+".tips.json")
}

func LoadState(sessionID string) State {
	s := State{LastShown: make(map[string]int)}
	data, err := os.ReadFile(StatePath(sessionID))
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	if s.LastShown == nil {
		s.LastShown = make(map[string]int)
	}
	return s
}

func (s State) Save(sessionID string) error {
	p := StatePath(sessionID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, _ := json.Marshal(s)
	return os.WriteFile(p, data, 0o644)
}
