package csvlog

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ccstats/internal/stats"
)

var Header = []string{
	"session_id", "start", "end", "duration_min", "model_primary",
	"model_breakdown", "turns", "words", "loc_delta",
	"total_tokens", "input_tokens", "output_tokens",
	"cache_read", "cache_create", "cost_usd", "burn_rate_tpm",
}

func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "ccstats", "sessions.csv")
}

func Upsert(s *stats.SessionStats) error {
	if s == nil || s.SessionID == "" {
		return fmt.Errorf("missing session id")
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	rows, _ := readAll(path)
	if len(rows) == 0 || !equalSlice(rows[0], Header) {
		rows = append([][]string{Header}, rows...)
	}

	breakdown := make(map[string]int)
	for k, m := range s.Models {
		breakdown[stats.ShortName(k)] += m.InputTokens + m.OutputTokens + m.CacheReadTokens + m.CacheCreationTokens
	}
	bjson, _ := json.Marshal(breakdown)

	row := []string{
		s.SessionID,
		formatTS(s.Start),
		formatTS(s.LastActivity),
		fmt.Sprintf("%.2f", s.DurationMin()),
		s.Model,
		string(bjson),
		fmt.Sprintf("%d", s.Turns),
		fmt.Sprintf("%d", s.Words),
		fmt.Sprintf("%d", s.LOCDelta),
		fmt.Sprintf("%d", s.TotalTokens()),
		fmt.Sprintf("%d", s.TotalInput),
		fmt.Sprintf("%d", s.TotalOutput),
		fmt.Sprintf("%d", s.TotalCacheRead),
		fmt.Sprintf("%d", s.TotalCacheCreate),
		fmt.Sprintf("%.4f", s.TotalCost),
		fmt.Sprintf("%.0f", s.BurnRate),
	}

	updated := false
	for i := 1; i < len(rows); i++ {
		if len(rows[i]) > 0 && rows[i][0] == s.SessionID {
			rows[i] = row
			updated = true
			break
		}
	}
	if !updated {
		rows = append(rows, row)
	}
	return writeAtomic(path, rows)
}

func formatTS(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func readAll(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	return r.ReadAll()
}

func writeAtomic(path string, rows [][]string) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		f.Close()
		return err
	}
	w.Flush()
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
