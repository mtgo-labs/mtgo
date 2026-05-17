package tg

import (
	"bytes"
	"encoding/binary"
	"io"
)

// TLObject is the interface implemented by all TL serializable types.
// It provides binary encoding and a unique constructor identifier.
type TLObject interface {
	// Encode writes the TL-encoded representation of the object to b.
	Encode(b *bytes.Buffer) error
	// ConstructorID returns the 32-bit TL constructor identifier.
	ConstructorID() uint32
}

// EncodeTLObject encodes obj into b using the TLObject.Encode method.
func EncodeTLObject(b *bytes.Buffer, obj TLObject) error {
	return obj.Encode(b)
}

// Registry maps TL constructor IDs to factory functions that decode the
// corresponding TLObject from a reader. Generated types register themselves
// during init.
var Registry = map[uint32]func(io.Reader) (TLObject, error){}

// ReadTLObject reads a TLObject from r by looking up the constructor ID in
// Registry and calling the associated factory function.
func ReadTLObject(r io.Reader) (TLObject, error) {
	var id uint32
	if err := binary.Read(r, binary.LittleEndian, &id); err != nil {
		return nil, err
	}
	constructor, ok := Registry[id]
	if !ok {
		return nil, &UnknownConstructorError{ID: id}
	}
	obj, err := constructor(r)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// UnknownConstructorError is returned when a TL constructor ID is not found
// in the global Registry.
type UnknownConstructorError struct {
	// ID is the unrecognized 32-bit constructor identifier.
	ID uint32
}

// Error returns a human-readable description of the unknown constructor.
func (e *UnknownConstructorError) Error() string {
	return "tg: unknown constructor ID 0x" + itox(e.ID)
}

func itox(v uint32) string {
	const hex = "0123456789abcdef"
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = hex[v&0xf]
		v >>= 4
	}
	return string(b)
}

type vectorTooLargeError struct {
	count uint32
}

func (e *vectorTooLargeError) Error() string {
	return "tg: vector too large"
}

const maxVectorElements uint32 = 100000

// CheckVectorCount rejects implausibly large TL vector lengths before allocating.
func CheckVectorCount(count uint32) error {
	if count > maxVectorElements {
		return &vectorTooLargeError{count: count}
	}
	return nil
}

func checkVectorCount(count uint32) error {
	return CheckVectorCount(count)
}
