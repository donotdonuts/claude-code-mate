package detail

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ccstats/internal/account"
	"ccstats/internal/render"
	"ccstats/internal/stats"
)

type model struct {
	sess    *stats.SessionStats
	win     *stats.WindowStats
	plan    render.Plan
	account *account.Account
	now     time.Time
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	headerStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	barFill     = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	barBg       = lipgloss.NewStyle().Faint(true)
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	overStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Claude Code — Session Detail"))
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("Current Session"))
	b.WriteString("\n")
	row(&b, "Session", short(m.sess.SessionID))
	row(&b, "Model", m.sess.Model)
	row(&b, "Started", m.sess.Start.Local().Format("Mon 15:04"))
	row(&b, "Duration", durationOf(m.sess.Duration()))
	row(&b, "Turns", fmt.Sprintf("%d", m.sess.Turns))
	row(&b, "Words sent", fmt.Sprintf("%d", m.sess.Words))
	row(&b, "Lines changed", fmt.Sprintf("%+d", m.sess.LOCDelta))
	row(&b, "Cost (this session)", fmt.Sprintf("$%.4f", m.sess.TotalCost))
	row(&b, "Context", fmt.Sprintf("%d / %d  (%.0f%%)", m.sess.LastCtxTokens, m.sess.ContextWindow(), m.sess.ContextPct()))
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Tokens (this session)"))
	b.WriteString("\n")
	row(&b, "Input", commas(m.sess.TotalInput))
	row(&b, "Output", commas(m.sess.TotalOutput))
	row(&b, "Cache read", commas(m.sess.TotalCacheRead))
	row(&b, "Cache create", commas(m.sess.TotalCacheCreate))
	row(&b, "Total", commas(m.sess.TotalTokens()))
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Per-Model Breakdown"))
	b.WriteString("\n")
	mus := make([]*stats.ModelUsage, 0, len(m.sess.Models))
	for _, v := range m.sess.Models {
		mus = append(mus, v)
	}
	sort.Slice(mus, func(i, j int) bool { return mus[i].Cost > mus[j].Cost })
	for _, mu := range mus {
		total := mu.InputTokens + mu.OutputTokens + mu.CacheReadTokens + mu.CacheCreationTokens
		line := fmt.Sprintf("  %-8s  %d turns · %s tok · $%.4f",
			stats.ShortName(mu.Model), mu.Turns, commas(total), mu.Cost)
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(headerStyle.Render(fmt.Sprintf("5-Hour Window  (%s plan, estimated)", m.plan.Name)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Authoritative figures only appear in the live statusline (Claude Code's payload)."))
	b.WriteString("\n")
	row(&b, "Window start", m.win.Anchor.Local().Format("15:04"))
	row(&b, "Resets at (est)", m.win.Reset.Local().Format("15:04"))

	tokPct := 0.0
	if m.plan.TokenLimit > 0 {
		tokPct = float64(m.win.TotalTokens) / float64(m.plan.TokenLimit) * 100
	}
	costPct := 0.0
	if m.plan.CostLimit > 0 {
		costPct = m.win.TotalCost / m.plan.CostLimit * 100
	}

	tokLine := fmt.Sprintf("%s / %s  %s", commas(m.win.TotalTokens), commas(m.plan.TokenLimit), bar(tokPct, 20))
	if tokPct > 100 {
		tokLine = warnStyle.Render(tokLine)
	}
	row(&b, "Tokens", tokLine)

	costLine := fmt.Sprintf("$%.2f / $%.2f  %s", m.win.TotalCost, m.plan.CostLimit, bar(costPct, 20))
	if m.win.TotalCost > m.plan.CostLimit {
		costLine = overStyle.Render(fmt.Sprintf("$%.2f / $%.2f  OVER by $%.2f", m.win.TotalCost, m.plan.CostLimit, m.win.TotalCost-m.plan.CostLimit))
	}
	row(&b, "Cost", costLine)
	row(&b, "Messages", fmt.Sprintf("%d", m.win.Messages))
	row(&b, "Burn rate", fmt.Sprintf("%.0f tok/min", m.win.BurnRate))

	winFreshInput := m.win.TotalInput + m.win.TotalCacheCreate
	winCacheTotal := m.win.TotalCacheRead + m.win.TotalCacheCreate
	hitPct := 0.0
	if denom := m.win.TotalCacheRead + winFreshInput; denom > 0 {
		hitPct = float64(m.win.TotalCacheRead) / float64(denom) * 100
	}
	row(&b, "Cache read (5h)", fmt.Sprintf("%s  (%.0f%% hit)", commas(m.win.TotalCacheRead), hitPct))
	row(&b, "Cache create (5h)", commas(m.win.TotalCacheCreate))
	row(&b, "Cache total (5h)", commas(winCacheTotal))

	until := time.Until(m.win.Reset)
	if until > 0 {
		row(&b, "Time to reset", durationOf(until))
		if m.win.BurnRate > 0 {
			projected := m.win.TotalTokens + int(m.win.BurnRate*until.Minutes())
			pp := 0.0
			if m.plan.TokenLimit > 0 {
				pp = float64(projected) / float64(m.plan.TokenLimit) * 100
			}
			projLine := fmt.Sprintf("%s tok (%.0f%%)", commas(projected), pp)
			if pp > 100 {
				projLine = warnStyle.Render(projLine + "  ⚠ projected to exceed plan")
			}
			row(&b, "Projected at reset", projLine)
		}
	}
	b.WriteString("\n")

	b.WriteString(dimStyle.Render("press q to quit"))
	return b.String()
}

func row(b *strings.Builder, label, value string) {
	b.WriteString(fmt.Sprintf("  %-22s %s\n", label+":", value))
}

func short(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:8] + "…"
}

func durationOf(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) - h*60
	return fmt.Sprintf("%dh %dm", h, m)
}

func commas(n int) string {
	if n < 0 {
		return "-" + commas(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var out strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteRune(',')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func bar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(float64(width) * pct / 100)
	return barFill.Render(strings.Repeat("█", filled)) + barBg.Render(strings.Repeat("░", width-filled))
}

func Run(transcriptPath string) error {
	sess, err := stats.Compute(transcriptPath)
	if err != nil {
		return err
	}
	win, _ := stats.ComputeWindow(time.Now())
	acct, _ := account.Load()
	plan := render.PlanFromAccount(acct)
	m := model{sess: sess, win: win, plan: plan, account: acct, now: time.Now()}

	if !isTTY(os.Stdout) {
		out := strings.TrimSuffix(m.View(), dimStyle.Render("press q to quit"))
		fmt.Print(strings.TrimRight(out, "\n"))
		fmt.Println()
		return nil
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func RunLatest() error {
	p, err := findLatestTranscript()
	if err != nil {
		return err
	}
	return Run(p)
}

func findLatestTranscript() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	pattern := filepath.Join(home, ".claude", "projects", "*", "*.jsonl")
	files, _ := filepath.Glob(pattern)
	var newest string
	var newestTime time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newest = f
			newestTime = info.ModTime()
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no transcripts found under ~/.claude/projects")
	}
	return newest, nil
}
