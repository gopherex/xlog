package sink

import "io"

type Discard struct{}

func NewDiscard() Discard { return Discard{} }

func (Discard) Write(p []byte) (int, error) { return io.Discard.Write(p) }
func (Discard) Sync() error                 { return nil }
func (Discard) Close() error                { return nil }
