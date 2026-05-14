package sink_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gopherex/xlog/pkg/sink"
)

func TestMultiWritesAll(t *testing.T) {
	var a bytes.Buffer
	var b bytes.Buffer
	w := sink.NewMulti(&a, &b)

	n, err := w.Write([]byte("log\n"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != len("log\n") {
		t.Fatalf("n = %d", n)
	}
	if a.String() != "log\n" || b.String() != "log\n" {
		t.Fatalf("outputs = %q %q", a.String(), b.String())
	}
}

func TestMultiReportsErrors(t *testing.T) {
	var out bytes.Buffer
	w := sink.NewMulti(&out, failingWriter{})

	n, err := w.Write([]byte("log\n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if n != 0 {
		t.Fatalf("n = %d", n)
	}
	if out.String() != "log\n" {
		t.Fatalf("successful writer did not receive data: %q", out.String())
	}
}

func TestWriterSyncClose(t *testing.T) {
	target := &syncCloseWriter{}
	w := sink.NewWriter(target)

	if err := w.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !target.synced || !target.closed {
		t.Fatalf("synced=%v closed=%v", target.synced, target.closed)
	}
}

func TestOpenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "app.log")
	w, err := sink.OpenFile(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}

	if _, err := w.Write([]byte("one\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != "one\n" {
		t.Fatalf("file = %q", string(got))
	}
}

func TestOpenFileAppendsExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	w, err := sink.OpenFile(path, sink.WithMaxSize(0))
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	if _, err := w.Write([]byte("new\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != "old\nnew\n" {
		t.Fatalf("file = %q", string(got))
	}
}

func TestOpenFileRotatesBySize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	clock := stepClock(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC))
	w, err := sink.OpenFile(path,
		sink.WithMaxSize(12),
		sink.WithMaxBackups(10),
		sink.WithClock(clock),
	)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}

	writeString(t, w, "12345\n")
	writeString(t, w, "67890\n")
	writeString(t, w, "abc\n")
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	active, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read active: %v", err)
	}
	if string(active) != "abc\n" {
		t.Fatalf("active = %q", string(active))
	}

	backups := glob(t, filepath.Join(dir, "app-*.log"))
	if len(backups) != 1 {
		t.Fatalf("backups = %v", backups)
	}
	rotated, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatalf("read rotated: %v", err)
	}
	if string(rotated) != "12345\n67890\n" {
		t.Fatalf("rotated = %q", string(rotated))
	}
}

func TestOpenFileRetention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	clock := stepClock(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC))
	w, err := sink.OpenFile(path,
		sink.WithMaxSize(4),
		sink.WithMaxBackups(2),
		sink.WithClock(clock),
	)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}

	for _, line := range []string{"aa\n", "bb\n", "cc\n", "dd\n", "ee\n"} {
		writeString(t, w, line)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	backups := glob(t, filepath.Join(dir, "app-*.log"))
	if len(backups) != 2 {
		t.Fatalf("backups = %v", backups)
	}
	if !strings.Contains(filepath.Base(backups[0]), "100002") ||
		!strings.Contains(filepath.Base(backups[1]), "100003") {
		t.Fatalf("unexpected retained backups: %v", backups)
	}
}

func TestOpenFileCompressesBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	clock := stepClock(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC))
	w, err := sink.OpenFile(path,
		sink.WithMaxSize(6),
		sink.WithMaxBackups(3),
		sink.WithCompress(true),
		sink.WithClock(clock),
	)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}

	writeString(t, w, "aa\n")
	writeString(t, w, "bb\n")
	writeString(t, w, "cc\n")
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	backups := glob(t, filepath.Join(dir, "app-*.log.gz"))
	if len(backups) != 1 {
		t.Fatalf("backups = %v", backups)
	}
	got := readGzip(t, backups[0])
	if got != "aa\nbb\n" {
		t.Fatalf("compressed = %q", got)
	}
	if plain := glob(t, filepath.Join(dir, "app-*.log")); len(plain) != 0 {
		t.Fatalf("plain backups still exist: %v", plain)
	}
}

func TestOpenFileAvoidsBackupNameCollisions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	now := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	w, err := sink.OpenFile(path,
		sink.WithMaxSize(4),
		sink.WithMaxBackups(10),
		sink.WithClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}

	for _, line := range []string{"aa\n", "bb\n", "cc\n", "dd\n"} {
		writeString(t, w, line)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	backups := glob(t, filepath.Join(dir, "app-*.log"))
	if len(backups) != 3 {
		t.Fatalf("backups = %v", backups)
	}
	if filepath.Base(backups[0]) == filepath.Base(backups[1]) ||
		filepath.Base(backups[1]) == filepath.Base(backups[2]) {
		t.Fatalf("backup name collision: %v", backups)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type syncCloseWriter struct {
	synced bool
	closed bool
}

func (w *syncCloseWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *syncCloseWriter) Sync() error {
	w.synced = true
	return nil
}
func (w *syncCloseWriter) Close() error {
	w.closed = true
	return nil
}

var _ io.Writer = (*syncCloseWriter)(nil)

func writeString(t *testing.T, w io.Writer, s string) {
	t.Helper()
	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatalf("write %q: %v", s, err)
	}
}

func glob(t *testing.T, pattern string) []string {
	t.Helper()
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %q: %v", pattern, err)
	}
	return files
}

func readGzip(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("new gzip reader: %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	return string(data)
}

func stepClock(start time.Time) func() time.Time {
	var i int
	return func() time.Time {
		t := start.Add(time.Duration(i) * time.Second)
		i++
		return t
	}
}
