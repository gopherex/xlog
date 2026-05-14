package jsoncore_test

import (
	"bytes"
	"testing"

	"github.com/gopherex/xlog/internal/jsoncore"
	"github.com/gopherex/xlog/pkg/core"
)

func TestWithTimeLayoutDoesNotReplaceCustomEncoder(t *testing.T) {
	encoder := fixedEncoder{}
	var out bytes.Buffer
	c := jsoncore.New(&out,
		jsoncore.WithEncoder(encoder),
		jsoncore.WithTimeLayout("ignored"),
	)

	if err := c.Write(core.Event{Level: core.InfoLevel, Message: "msg"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if out.String() != `{"custom":true}`+"\n" {
		t.Fatalf("out = %q", out.String())
	}
}

type fixedEncoder struct{}

func (fixedEncoder) Encode(dst []byte, event core.Event) []byte {
	return append(dst, `{"custom":true}`...)
}
