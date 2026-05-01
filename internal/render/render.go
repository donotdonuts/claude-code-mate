package render

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"ccstats/internal/account"
	"ccstats/internal/stats"
	"ccstats/internal/tips"
)

const (
	dim   = "\x1b[2m"
	reset = "\x1b[0m"
)

type Limit struct {
	UsedPercentage float64
	ResetsAt       int64
}

type View struct {
	Model         string
	ModelID       string
	Sess          *stats.SessionStats
	Account       *account.Account
	CtxSize       int
	CtxUsed       int
	CtxPct        *float64
	FiveHour      *Limit
	SevenDay      *Limit
	LinesAdded    int
	LinesRemoved  int
	HasLineCounts bool
	Tip           *tips.Tip
}

// Statusline renders a multi-line statusline. Empty lines (e.g. model
// breakdown when only one model is in use) are skipped entirely so the
// output stays compact.
func Statusline(v View) string {
	var b strings.Builder
	b.WriteString(dim)

	b.WriteString(line1(v))

	if seg := line2(v); seg != "" {
		b.WriteByte('\n')
		b.WriteString(seg)
	}
	if seg := line3(v); seg != "" {
		b.WriteByte('\n')
		b.WriteString(seg)
	}
	if seg := line4(v); seg != "" {
		b.WriteByte('\n')
		b.WriteString(seg)
	}

	b.WriteString(reset)

	if v.Tip != nil {
		b.WriteByte('\n')
		b.WriteString("tip: ")
		b.WriteString(v.Tip.Message)
	}
	return b.String()
}

// stripContextSuffix removes a trailing parenthetical that mentions "context"
// (e.g. "Opus 4.7 (1M context)" → "Opus 4.7"). The context size is already
// shown in the ctx segment so this suffix is redundant.
var contextSuffixRe = regexp.MustCompile(`\s*\([^)]*context[^)]*\)\s*$`)

func stripContextSuffix(s string) string {
	return contextSuffixRe.ReplaceAllString(s, "")
}

func resolveModel(v View) string {
	model := v.Model
	if model == "" {
		model = humanModel(firstNonEmpty(v.ModelID, v.Sess.Model))
	}
	if model == "" {
		return "claude"
	}
	return stripContextSuffix(model)
}

func line1(v View) string {
	model := resolveModel(v)

	ctxSize := v.CtxSize
	if ctxSize == 0 {
		ctxSize = v.Sess.ContextWindow()
	}
	ctxUsed := v.CtxUsed
	if ctxUsed == 0 {
		ctxUsed = v.Sess.LastCtxTokens
	}
	var pct float64
	switch {
	case v.CtxPct != nil:
		pct = *v.CtxPct
	case ctxSize > 0 && ctxUsed > 0:
		pct = float64(ctxUsed) / float64(ctxSize) * 100
	default:
		pct = v.Sess.ContextPct()
	}
	ctx := fmt.Sprintf("ctx %d%% (%s/%s)", int(pct+0.5), humanTokens(ctxUsed), humanTokens(ctxSize))

	planName := ""
	if v.Account != nil {
		planName = v.Account.PlanName
	}
	if planName == "" {
		planName = "—"
	}
	var usage, resetStr string
	if v.FiveHour == nil {
		usage = fmt.Sprintf("Usage — (%s)", planName)
		resetStr = "Resets —"
	} else {
		usage = fmt.Sprintf("Usage %d%% (%s)", int(v.FiveHour.UsedPercentage+0.5), planName)
		resetStr = "Resets —"
		if v.FiveHour.ResetsAt > 0 {
			resetStr = "Resets " + time.Unix(v.FiveHour.ResetsAt, 0).Local().Format("3:04pm")
		}
	}
	return fmt.Sprintf("%s · %s · %s · %s", model, ctx, usage, resetStr)
}

func line2(v View) string {
	read := v.Sess.TotalCacheRead
	create := v.Sess.TotalCacheCreate
	freshInput := v.Sess.TotalInput + create
	denom := read + freshInput
	if denom == 0 {
		return "cache 0% hit · 0 read · 0 create"
	}
	hit := float64(read) / float64(denom) * 100
	return fmt.Sprintf("cache %d%% hit · %s read · %s create",
		int(hit+0.5), humanShort(read), humanShort(create))
}

func line3(v View) string {
	pieces := []string{
		humanDuration(v.Sess.Duration()),
		fmt.Sprintf("%d turns", v.Sess.Turns),
		fmt.Sprintf("%d words", v.Sess.Words),
	}
	if v.HasLineCounts && (v.LinesAdded != 0 || v.LinesRemoved != 0) {
		pieces = append(pieces, fmt.Sprintf("+%s/-%s LOC",
			humanShort(v.LinesAdded), humanShort(v.LinesRemoved)))
	} else {
		pieces = append(pieces, fmt.Sprintf("%s LOC", humanShort(v.Sess.LOCDelta)))
	}
	pieces = append(pieces, fmt.Sprintf("%s tok", humanShort(v.Sess.TotalTokens())))
	pieces = append(pieces, fmt.Sprintf("%s tok/min", humanShort(int(v.Sess.BurnRate))))
	return strings.Join(pieces, " · ")
}

func line4(v View) string {
	models := v.Sess.ModelBreakdown()
	if len(models) <= 1 {
		return ""
	}
	parts := make([]string, 0, len(models))
	for _, m := range models {
		parts = append(parts, fmt.Sprintf("%s %d%%", m.Model, int(m.Pct+0.5)))
	}
	return strings.Join(parts, " · ")
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func humanModel(m string) string {
	if m == "" {
		return "claude"
	}
	low := strings.ToLower(m)
	var family string
	switch {
	case strings.Contains(low, "opus"):
		family = "Opus"
	case strings.Contains(low, "sonnet"):
		family = "Sonnet"
	case strings.Contains(low, "haiku"):
		family = "Haiku"
	default:
		return m
	}
	re := regexp.MustCompile(`(\d+)-(\d+)`)
	if v := re.FindStringSubmatch(low); v != nil {
		return fmt.Sprintf("%s %s.%s", family, v[1], v[2])
	}
	return family
}

func humanDuration(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) - h*60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func humanTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	if n%1_000_000 == 0 {
		return fmt.Sprintf("%dM", n/1_000_000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1e6)
}

func humanShort(n int) string {
	if n < 0 {
		return "-" + humanShort(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1e6)
}
