package xlog

import (
	"errors"
	"net/url"
	"strings"

	xfield "github.com/gopherex/xlog/pkg/field"
)

// Redactor renders one layer of an error chain when it recognizes that
// layer's type: ok reports it did. The engine keeps descending, so a child
// error's text inside the returned text is rendered in turn.
type Redactor func(err error) (text string, ok bool)

// NewErrorRenderer returns a renderer for telemetry: it walks the error chain
// and lets the first matching Redactor rewrite each layer, keeping every
// wrapper's own text (fmt.Errorf prefixes, errors.Join lines). It only
// renders text; the error itself is untouched, so errors.Is/As keep working.
func NewErrorRenderer(rs ...Redactor) func(error) string {
	return func(err error) string { return renderError(err, rs) }
}

func renderError(err error, rs []Redactor) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	for _, r := range rs {
		if t, ok := r(err); ok {
			text = t
			break
		}
	}
	// Replace each child's text by its rendering wherever the wrapper embeds
	// it (prefix, suffix or middle; one line per child of errors.Join).
	for _, child := range children(err) {
		raw := child.Error()
		if rendered := renderError(child, rs); rendered != raw {
			text = strings.Replace(text, raw, rendered, 1)
		}
	}
	return text
}

func children(err error) []error {
	switch u := err.(type) {
	case interface{ Unwrap() []error }:
		return u.Unwrap()
	case interface{ Unwrap() error }:
		if inner := u.Unwrap(); inner != nil {
			return []error{inner}
		}
	}
	return nil
}

// RedactURL renders a *url.Error as its operation and the URL's scheme and
// host only: no user info, path or query (presigned signatures, tokens). The
// underlying cause is kept and rendered in turn.
func RedactURL(err error) (string, bool) {
	switch e := err.(type) {
	case *url.Error:
		target := e.URL
		if u, perr := url.Parse(e.URL); perr == nil {
			target = u.Scheme + "://" + u.Host
		}
		cause := ""
		if e.Err != nil {
			cause = e.Err.Error()
		}
		return e.Op + ` "` + target + `": ` + cause, true
	}
	return "", false
}

// renderedError carries a rendered text for encoders while unwrapping to the
// original, so errors.Is/As on a logged field still see the real error.
type renderedError struct {
	err  error
	text string
}

func (e renderedError) Error() string { return e.text }
func (e renderedError) Unwrap() error { return e.err }

// errorTexts is the lazily rendered value of ErrorChain / Errors: the logger
// turns it into strings at write time, through its renderer.
type errorTexts []error

// WithErrorRenderer renders every error a logger writes through r: Err/Error
// fields, errors in Any, ErrorChain, Errors, With and context fields, and the
// root WithFields. Set it once on the root; derived loggers inherit it. Use
// NewErrorRenderer to build one from Redactors.
func WithErrorRenderer(r func(error) string) Option {
	return func(c *Config) { c.ErrorRenderer = r }
}

// renderFields returns fields with errors rendered: by render when set, and
// ErrorChain / Errors always resolved to strings. fields is not modified.
func renderFields(fields []Field, render func(error) string) []Field {
	var out []Field
	for i, f := range fields {
		next, changed := renderField(f, render)
		if !changed {
			continue
		}
		if out == nil {
			out = make([]Field, len(fields))
			copy(out, fields)
		}
		out[i] = next
	}
	if out == nil {
		return fields
	}
	return out
}

func renderField(f Field, render func(error) string) (Field, bool) {
	switch v := f.AnyValue().(type) {
	case errorTexts:
		texts := make([]string, 0, len(v))
		for _, err := range v {
			texts = append(texts, textOf(err, render))
		}
		return Any(f.Key, texts), true
	case error:
		if render == nil {
			return f, false
		}
		rendered := renderedError{err: v, text: render(v)}
		if f.Kind == xfield.ErrorKind {
			return Error(f.Key, rendered), true
		}
		return Any(f.Key, rendered), true
	}
	return f, false
}

func textOf(err error, render func(error) string) string {
	if render == nil {
		return err.Error()
	}
	return render(err)
}

// errChain lists every layer of err for ErrorChain.
func errChain(err error) errorTexts {
	var chain errorTexts
	for current := err; current != nil; current = errors.Unwrap(current) {
		chain = append(chain, current)
	}
	return chain
}
