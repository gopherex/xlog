package core

// Encoder converts an event to bytes. Implementations may reuse dst.
type Encoder interface {
	Encode(dst []byte, event Event) []byte
}
