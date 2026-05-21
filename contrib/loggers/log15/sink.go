package log15

import (
	"fmt"

	l15 "gopkg.in/inconshreveable/log15.v2"

	"github.com/gopherex/xlog"
)

// NewSinkHandler wraps an xlog.Logger as a log15.Handler.
//
//	l := l15.New()
//	l.SetHandler(log15adapter.NewSinkHandler(xl))
func NewSinkHandler(l *xlog.Logger) l15.Handler {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return l15.FuncHandler(func(r *l15.Record) error {
		fields := make([]xlog.Field, 0, len(r.Ctx)/2)
		for i := 0; i+1 < len(r.Ctx); i += 2 {
			key := fmt.Sprint(r.Ctx[i])
			fields = append(fields, xlog.Any(key, r.Ctx[i+1]))
		}
		l.Log(fromLog15Level(r.Lvl), r.Msg, fields...)
		return nil
	})
}

func fromLog15Level(lv l15.Lvl) xlog.Level {
	switch lv {
	case l15.LvlDebug:
		return xlog.DebugLevel
	case l15.LvlInfo:
		return xlog.InfoLevel
	case l15.LvlWarn:
		return xlog.WarnLevel
	case l15.LvlCrit:
		return xlog.CriticalLevel
	}
	return xlog.ErrorLevel
}
