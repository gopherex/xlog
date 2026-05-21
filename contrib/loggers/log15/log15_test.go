package log15_test

import (
	"bytes"
	"strings"
	"testing"

	l15 "gopkg.in/inconshreveable/log15.v2"

	"github.com/gopherex/xlog"
	l15adapter "github.com/gopherex/xlog/contrib/loggers/log15"
)

func TestLog15AdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	l := l15.New()
	l.SetHandler(l15.StreamHandler(&buf, l15.LogfmtFormat()))

	logger := xlog.New(l15adapter.New(l)).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	got := buf.String()
	if !strings.Contains(got, "msg=started") ||
		!strings.Contains(got, "service=api") ||
		!strings.Contains(got, "request_id=r1") {
		t.Fatalf("out = %q", got)
	}
}

func TestLog15AdapterLevelMapping(t *testing.T) {
	cases := []struct {
		level   xlog.Level
		wantLvl string
	}{
		{xlog.TraceLevel, "lvl=dbug"},
		{xlog.DebugLevel, "lvl=dbug"},
		{xlog.InfoLevel, "lvl=info"},
		{xlog.WarnLevel, "lvl=warn"},
		{xlog.ErrorLevel, "lvl=eror"},
		{xlog.CriticalLevel, "lvl=crit"},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		l := l15.New()
		l.SetHandler(l15.StreamHandler(&buf, l15.LogfmtFormat()))

		logger := xlog.New(l15adapter.New(l))
		logger.Log(tc.level, "msg")

		got := buf.String()
		if !strings.Contains(got, tc.wantLvl) {
			t.Fatalf("level %v: out = %q, want %q", tc.level, got, tc.wantLvl)
		}
	}
}
