package xlog_test

import (
	"io"
	"sync"
	"testing"

	"github.com/gopherex/xlog"
)

// With on a logger that is writing concurrently must not race (run with
// -race).
func TestWithWhileWriting(t *testing.T) {
	for name, logger := range map[string]*xlog.Logger{
		"json":    xlog.NewJSON(xlog.WithWriter(io.Discard)),
		"console": xlog.NewConsole(xlog.WithWriter(io.Discard)),
	} {
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup

			for i := range 4 {
				wg.Go(func() {
					for range 200 {
						logger.Info("write", xlog.Int("i", i))
					}
				})
				wg.Go(func() {
					for range 200 {
						logger.With(xlog.Int("child", i)).Info("child")
					}
				})
			}

			wg.Wait()
		})
	}
}
