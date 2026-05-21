// Package aws routes AWS SDK for Go v2 logs into xlog.
//
// The SDK logs through smithy-go's logging.Logger, exposed as
// aws.Config.Logger. New returns an adapter that also implements
// logging.ContextLogger, so the SDK's per-request context (carrying e.g. OTel
// trace context) is forwarded into xlog:
//
//	cfg, _ := config.LoadDefaultConfig(ctx,
//		config.WithLogger(awslog.New(xl)),
//		config.WithClientLogMode(aws.LogRequest|aws.LogRetries),
//	)
package aws

import (
	"context"
	"fmt"

	"github.com/aws/smithy-go/logging"

	"github.com/gopherex/xlog"
)

// New wraps an xlog.Logger as a smithy-go logging.Logger for the AWS SDK v2.
// Set the result as aws.Config.Logger (e.g. via config.WithLogger).
func New(l *xlog.Logger) logging.Logger {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return &logger{ctxLog: l.Ctx()}
}

// logger implements logging.Logger and logging.ContextLogger.
type logger struct {
	ctxLog *xlog.ContextLogger
	ctx    context.Context
}

func (l *logger) Logf(classification logging.Classification, format string, v ...any) {
	ctx := l.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	l.ctxLog.Log(ctx, fromClassification(classification), fmt.Sprintf(format, v...))
}

// WithContext returns a logger that forwards ctx to xlog on each Logf call.
func (l *logger) WithContext(ctx context.Context) logging.Logger {
	next := *l
	next.ctx = ctx
	return &next
}

// fromClassification maps smithy-go classifications to xlog levels. The SDK
// emits only Debug and Warn; anything else falls back to Info.
func fromClassification(c logging.Classification) xlog.Level {
	switch c {
	case logging.Debug:
		return xlog.DebugLevel
	case logging.Warn:
		return xlog.WarnLevel
	}
	return xlog.InfoLevel
}
