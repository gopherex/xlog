package sink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// maxSinkLineBytes caps the in-memory buffer for partial NDJSON lines. A line
// without a terminating '\n' that grows past this size is dropped to avoid
// unbounded memory growth from a misbehaving producer.
const maxSinkLineBytes = 1 << 20 // 1 MiB

// NewPretty wraps w with an NDJSON pretty-printer.
//
// Each newline-delimited JSON record is reformatted into a single human-readable
// line with a dimmed timestamp, a colored level, the message, and remaining
// fields as key=value pairs. Lines that fail to parse as JSON pass through
// unchanged. Safe for concurrent use.
//
// Drop-in for any NDJSON producer — jsoncore, the zap/zerolog/slog contribs,
// or any other backend that writes JSON-per-line.
func NewPretty(w io.Writer) io.Writer {
	return &prettyWriter{out: w}
}

type prettyWriter struct {
	mu  sync.Mutex
	out io.Writer
	buf bytes.Buffer
}

func (p *prettyWriter) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(b)
	p.buf.Write(b)
	for {
		data := p.buf.Bytes()
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			if p.buf.Len() > maxSinkLineBytes {
				p.buf.Reset()
			}
			break
		}
		line := append([]byte(nil), data[:i]...)
		p.buf.Next(i + 1)
		if err := p.formatLine(line); err != nil {
			return n, err
		}
	}
	return n, nil
}

func (p *prettyWriter) formatLine(line []byte) error {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		_, err := p.out.Write([]byte{'\n'})
		return err
	}

	var rec map[string]any
	if err := json.Unmarshal(line, &rec); err != nil {
		// passthrough non-JSON
		if _, err := p.out.Write(line); err != nil {
			return err
		}
		_, err := p.out.Write([]byte{'\n'})
		return err
	}

	ts := pickString(rec, "time", "ts", "timestamp", "@timestamp")
	level := pickString(rec, "level", "lvl", "@level", "severity")
	msg := pickString(rec, "msg", "message", "@message")
	logger := pickString(rec, "logger")

	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	if ts != "" {
		sb.WriteString(dim(ts))
		sb.WriteByte(' ')
	}
	sb.WriteString(colorLevel(level))
	sb.WriteByte(' ')
	if logger != "" {
		sb.WriteString(dim("[" + logger + "]"))
		sb.WriteByte(' ')
	}
	sb.WriteString(msg)
	for _, k := range keys {
		sb.WriteByte(' ')
		sb.WriteString(dim(k + "="))
		appendValue(&sb, rec[k])
	}
	sb.WriteByte('\n')

	_, err := io.WriteString(p.out, sb.String())
	return err
}

func pickString(rec map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := rec[k]; ok {
			delete(rec, k)
			s, _ := v.(string)
			return s
		}
	}
	return ""
}

func appendValue(sb *strings.Builder, v any) {
	switch t := v.(type) {
	case nil:
		sb.WriteString("null")
	case string:
		if strings.ContainsAny(t, ` "=`) {
			fmt.Fprintf(sb, "%q", t)
		} else {
			sb.WriteString(t)
		}
	case bool, float64:
		fmt.Fprintf(sb, "%v", t)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			fmt.Fprintf(sb, "%v", t)
			return
		}
		sb.Write(b)
	}
}

const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiBold   = "\x1b[1m"
)

func dim(s string) string { return ansiDim + s + ansiReset }

func colorLevel(level string) string {
	low := strings.ToLower(level)
	label := strings.ToUpper(level)
	if label == "" {
		label = "INFO"
	}
	if len(label) < 5 {
		label = label + strings.Repeat(" ", 5-len(label))
	}
	switch low {
	case "debug":
		return ansiCyan + label + ansiReset
	case "warn", "warning":
		return ansiYellow + label + ansiReset
	case "error", "err", "fatal", "panic":
		return ansiRed + ansiBold + label + ansiReset
	}
	return ansiGreen + label + ansiReset
}
