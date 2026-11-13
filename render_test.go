package xlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const signed = "http://127.0.0.1:1/bucket/key?X-Amz-Signature=secret-sig&token=secret-token"

// realURLError is what an HTTP client returns for a signed URL.
func realURLError(t testing.TB) error {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, signed, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req) // a refused dial: nothing leaves the host
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected a dial error")
	}
	var u *url.Error
	if !errors.As(err, &u) {
		t.Fatalf("not a *url.Error: %T", err)
	}
	return err
}

func TestErrorRenderer(t *testing.T) {
	t.Parallel()
	urlErr := realURLError(t)
	render := NewErrorRenderer(RedactURL)
	for _, tc := range []struct {
		name string
		err  error
		want string // a substring that must remain
	}{
		{name: "url error", err: urlErr, want: `Get "http://127.0.0.1:1": dial tcp`},
		{name: "wrapped", err: fmt.Errorf("download avatar: %w", urlErr), want: `download avatar: Get "http://127.0.0.1:1"`},
		{name: "wrapped in the middle", err: fmt.Errorf("%w (attempt 2)", urlErr), want: `(attempt 2)`},
		{name: "joined", err: errors.Join(errors.New("first"), urlErr), want: "first\nGet \"http://127.0.0.1:1\""},
		{name: "unrelated", err: errors.New("plain"), want: "plain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := render(tc.err)
			if strings.Contains(got, "secret") || strings.Contains(got, "/bucket") {
				t.Fatalf("leaked: %q", got)
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("render = %q, want it to contain %q", got, tc.want)
			}
		})
	}
	if render(nil) != "" {
		t.Fatal("nil renders empty")
	}
}

// Every way an error reaches the logger is rendered; the logged error still
// unwraps to the original.
func TestLoggerRendersErrors(t *testing.T) {
	t.Parallel()
	urlErr := realURLError(t)
	var buf bytes.Buffer
	l := NewJSON(WithWriter(&buf), WithErrorRenderer(NewErrorRenderer(RedactURL)),
		WithFields(Error("root_err", urlErr)))
	ctx := ContextWithFields(context.Background(), Error("ctx_err", urlErr))
	l.With(Error("with_err", urlErr)).Ctx().Info(ctx, "fetch failed",
		Err(urlErr),
		Any("any_err", urlErr),
		ErrorChain(fmt.Errorf("load: %w", urlErr)),
		Errors("errs", []error{urlErr, errors.New("other")}),
	)
	slog.New(NewSlogHandler(l)).With("slog_with", urlErr).Error("via slog", "slog_err", urlErr)
	l.Check(ErrorLevel, "checked").Write(Err(urlErr))
	out := buf.String()
	if strings.Contains(out, "secret") || strings.Contains(out, "/bucket") {
		t.Fatalf("leaked into the log: %s", out)
	}
	for _, key := range []string{"root_err", "ctx_err", "with_err", `"error"`, "any_err", "error_chain", "errs",
		"slog_with", "slog_err", "checked"} {
		if !strings.Contains(out, key) {
			t.Fatalf("field %s missing: %s", key, out)
		}
	}

	r := renderedError{err: urlErr, text: "x"}
	var u *url.Error
	if !errors.Is(r, urlErr) || !errors.As(r, &u) {
		t.Fatal("a rendered error must unwrap to the original")
	}
}

// Without a renderer the output is what it always was.
func TestLoggerWithoutRenderer(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	NewJSON(WithWriter(&buf)).Info("x",
		ErrorChain(fmt.Errorf("load: %w", errors.New("inner"))),
		Errors("errs", []error{errors.New("a"), nil, errors.New("b")}))
	out := buf.String()
	if !strings.Contains(out, `"error_chain":["load: inner","inner"]`) || !strings.Contains(out, `"errs":["a","b"]`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func BenchmarkErrorRenderer(b *testing.B) {
	err := fmt.Errorf("download: %w", realURLError(b))
	render := NewErrorRenderer(RedactURL)
	for b.Loop() {
		_ = render(err)
	}
}
