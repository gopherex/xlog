package aws_test

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/logging"

	"github.com/gopherex/xlog"
	awsadapter "github.com/gopherex/xlog/contrib/libs/aws"
	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

// captureCore records the last event written, for inspecting level/message/ctx.
type captureCore struct{ last core.Event }

func (c *captureCore) Enabled(core.Level) bool      { return true }
func (c *captureCore) Write(e core.Event) error     { c.last = e; return nil }
func (c *captureCore) With([]field.Field) core.Core { return c }
func (c *captureCore) Sync() error                  { return nil }

func TestLogfMapsClassificationAndFormats(t *testing.T) {
	cc := &captureCore{}
	lg := awsadapter.New(xlog.New(cc))

	cases := []struct {
		class logging.Classification
		want  core.Level
	}{
		{logging.Debug, xlog.DebugLevel},
		{logging.Warn, xlog.WarnLevel},
		{logging.Classification("OTHER"), xlog.InfoLevel},
	}
	for _, tc := range cases {
		lg.Logf(tc.class, "took %d ms", 42)
		if cc.last.Level != tc.want {
			t.Fatalf("class %q -> level %v, want %v", tc.class, cc.last.Level, tc.want)
		}
		if cc.last.Message != "took 42 ms" {
			t.Fatalf("message = %q", cc.last.Message)
		}
	}
}

type ctxKey struct{}

func TestWithContextForwardsContext(t *testing.T) {
	cc := &captureCore{}
	lg := awsadapter.New(xlog.New(cc))

	cl, ok := lg.(logging.ContextLogger)
	if !ok {
		t.Fatal("adapter does not implement logging.ContextLogger")
	}

	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-123")
	cl.WithContext(ctx).Logf(logging.Debug, "hi")

	if cc.last.Ctx == nil || cc.last.Ctx.Value(ctxKey{}) != "trace-123" {
		t.Fatalf("event ctx = %v, want value trace-123", cc.last.Ctx)
	}
}

func TestNilLoggerSafe(t *testing.T) {
	awsadapter.New(nil).Logf(logging.Warn, "no panic")
}
