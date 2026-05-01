package transcript

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"time"
)

type Entry struct {
	Type        string          `json:"type"`
	UUID        string          `json:"uuid"`
	ParentUUID  string          `json:"parentUuid"`
	Timestamp   string          `json:"timestamp"`
	SessionID   string          `json:"sessionId"`
	CWD         string          `json:"cwd"`
	GitBranch   string          `json:"gitBranch"`
	Version     string          `json:"version"`
	Message     json.RawMessage `json:"message"`
	IsSidechain bool            `json:"isSidechain"`
	ToolUseID   string          `json:"toolUseID"`
	Subtype     string          `json:"subtype"`
	IsError     bool            `json:"isError"`
}

type Message struct {
	Model   string          `json:"model"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Usage   *Usage          `json:"usage,omitempty"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

type ContentBlock struct {
	Type    string          `json:"type"`
	Text    string          `json:"text,omitempty"`
	Name    string          `json:"name,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	IsError bool            `json:"is_error,omitempty"`
}

func (e *Entry) Time() time.Time {
	t, err := time.Parse(time.RFC3339Nano, e.Timestamp)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, e.Timestamp)
	}
	return t
}

func Iterate(r io.Reader, fn func(*Entry) error) error {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1024*1024), 32*1024*1024)
	for s.Scan() {
		var e Entry
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			continue
		}
		if err := fn(&e); err != nil {
			return err
		}
	}
	return s.Err()
}

func IterateFile(path string, fn func(*Entry) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Iterate(f, fn)
}
