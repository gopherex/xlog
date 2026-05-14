package sink

import (
	"io"
	"os"
)

func Stdout() io.Writer { return os.Stdout }
func Stderr() io.Writer { return os.Stderr }
