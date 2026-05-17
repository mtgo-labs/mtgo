package tg

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// ReadInt reads a 32-bit little-endian unsigned integer from r.
func ReadInt(r io.Reader) uint32 {
	var v uint32
	_ = binary.Read(r, binary.LittleEndian, &v)
	return v
}

// ReadIntErr reads a 32-bit little-endian unsigned integer from r, returning
// an error if the read fails.
func ReadIntErr(r io.Reader) (uint32, error) {
	var v uint32
	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

// WriteInt writes a 32-bit little-endian unsigned integer to b.
func WriteInt(b *bytes.Buffer, v uint32) {
	_ = binary.Write(b, binary.LittleEndian, v)
}

// ReadLong reads a 64-bit little-endian signed integer from r.
func ReadLong(r io.Reader) int64 {
	var v int64
	_ = binary.Read(r, binary.LittleEndian, &v)
	return v
}

// WriteLong writes a 64-bit little-endian signed integer to b.
func WriteLong(b *bytes.Buffer, v int64) {
	_ = binary.Write(b, binary.LittleEndian, v)
}

// ReadInt128 reads a 128-bit (16-byte) value from r.
func ReadInt128(r io.Reader) [16]byte {
	var v [16]byte
	_, _ = io.ReadFull(r, v[:])
	return v
}

// WriteInt128 writes a 128-bit (16-byte) value to b.
func WriteInt128(b *bytes.Buffer, v [16]byte) {
	_, _ = b.Write(v[:])
}

// ReadInt256 reads a 256-bit (32-byte) value from r.
func ReadInt256(r io.Reader) [32]byte {
	var v [32]byte
	_, _ = io.ReadFull(r, v[:])
	return v
}

// WriteInt256 writes a 256-bit (32-byte) value to b.
func WriteInt256(b *bytes.Buffer, v [32]byte) {
	_, _ = b.Write(v[:])
}

// ReadDouble reads a 64-bit IEEE 754 float from r in little-endian order.
func ReadDouble(r io.Reader) float64 {
	var v uint64
	_ = binary.Read(r, binary.LittleEndian, &v)
	return math.Float64frombits(v)
}

// WriteDouble writes a 64-bit IEEE 754 float to b in little-endian order.
func WriteDouble(b *bytes.Buffer, v float64) {
	_ = binary.Write(b, binary.LittleEndian, math.Float64bits(v))
}

const (
	// BoolTrueID is the TL constructor ID for boolTrue.
	BoolTrueID uint32 = 0x997277B5
	// BoolFalseID is the TL constructor ID for boolFalse.
	BoolFalseID uint32 = 0xBC799737
)

// ReadBool reads a TL-encoded boolean from r.
func ReadBool(r io.Reader) bool {
	var id uint32
	_ = binary.Read(r, binary.LittleEndian, &id)
	return id == BoolTrueID
}

// WriteBool writes a TL-encoded boolean to b.
func WriteBool(b *bytes.Buffer, v bool) {
	if v {
		_ = binary.Write(b, binary.LittleEndian, BoolTrueID)
	} else {
		_ = binary.Write(b, binary.LittleEndian, BoolFalseID)
	}
}

// ReadBytes reads a TL-encoded byte string from r, including padding.
func ReadBytes(r io.Reader) []byte {
	var first [1]byte
	_, _ = io.ReadFull(r, first[:])
	length := int(first[0])
	headerLen := 1

	var data []byte
	if length <= 253 {
		if length == 0 {
			data = nil
		} else {
			data = make([]byte, length)
			_, _ = io.ReadFull(r, data)
		}
	} else {
		var lo [3]byte
		_, _ = io.ReadFull(r, lo[:])
		length = int(lo[0]) | int(lo[1])<<8 | int(lo[2])<<16
		headerLen = 4
		data = make([]byte, length)
		_, _ = io.ReadFull(r, data)
	}

	padding := (4 - (length+headerLen)%4) % 4
	if padding > 0 {
		discard := make([]byte, padding)
		_, _ = io.ReadFull(r, discard)
	}
	return data
}

// WriteBytes writes a TL-encoded byte string to b, including padding.
func WriteBytes(b *bytes.Buffer, v []byte) {
	length := 0
	if v != nil {
		length = len(v)
	}
	headerLen := 1
	if length <= 253 {
		b.WriteByte(byte(length))
	} else {
		headerLen = 4
		b.WriteByte(254)
		b.WriteByte(byte(length))
		b.WriteByte(byte(length >> 8))
		b.WriteByte(byte(length >> 16))
	}
	if length > 0 {
		b.Write(v)
	}
	padding := (4 - (length+headerLen)%4) % 4
	if padding > 0 {
		for i := 0; i < padding; i++ {
			b.WriteByte(0)
		}
	}
}

// ReadString reads a TL-encoded string from r.
func ReadString(r io.Reader) string {
	data := ReadBytes(r)
	return string(data)
}

// WriteString writes a TL-encoded string to b.
func WriteString(b *bytes.Buffer, v string) {
	WriteBytes(b, []byte(v))
}

const vectorBareID uint32 = 0x1cb5c415

// ReadVectorInt reads a TL-encoded vector of int32 values from r.
func ReadVectorInt(r io.Reader) []int32 {
	ReadInt(r)
	count := ReadInt(r)
	if checkVectorCount(count) != nil {
		return nil
	}
	result := make([]int32, count)
	for i := range result {
		result[i] = int32(ReadInt(r))
	}
	return result
}

// WriteVectorInt writes a TL-encoded vector of int32 values to b.
func WriteVectorInt(b *bytes.Buffer, v []int32) {
	WriteInt(b, vectorBareID)
	WriteInt(b, uint32(len(v)))
	for _, item := range v {
		WriteInt(b, uint32(item))
	}
}

// ReadVectorLong reads a TL-encoded vector of int64 values from r.
func ReadVectorLong(r io.Reader) []int64 {
	ReadInt(r)
	count := ReadInt(r)
	if checkVectorCount(count) != nil {
		return nil
	}
	result := make([]int64, count)
	for i := range result {
		result[i] = ReadLong(r)
	}
	return result
}

// WriteVectorLong writes a TL-encoded vector of int64 values to b.
func WriteVectorLong(b *bytes.Buffer, v []int64) {
	WriteInt(b, vectorBareID)
	WriteInt(b, uint32(len(v)))
	for _, item := range v {
		WriteLong(b, item)
	}
}

// ReadVectorString reads a TL-encoded vector of strings from r.
func ReadVectorString(r io.Reader) []string {
	ReadInt(r)
	count := ReadInt(r)
	if checkVectorCount(count) != nil {
		return nil
	}
	result := make([]string, count)
	for i := range result {
		result[i] = ReadString(r)
	}
	return result
}

// WriteVectorString writes a TL-encoded vector of strings to b.
func WriteVectorString(b *bytes.Buffer, v []string) {
	WriteInt(b, vectorBareID)
	WriteInt(b, uint32(len(v)))
	for _, item := range v {
		WriteString(b, item)
	}
}

// ReadVectorBytes reads a TL-encoded vector of byte slices from r.
func ReadVectorBytes(r io.Reader) [][]byte {
	ReadInt(r)
	count := ReadInt(r)
	if checkVectorCount(count) != nil {
		return nil
	}
	result := make([][]byte, count)
	for i := range result {
		result[i] = ReadBytes(r)
	}
	return result
}

// WriteVectorBytes writes a TL-encoded vector of byte slices to b.
func WriteVectorBytes(b *bytes.Buffer, v [][]byte) {
	WriteInt(b, vectorBareID)
	WriteInt(b, uint32(len(v)))
	for _, item := range v {
		WriteBytes(b, item)
	}
}

// TLBool is a bool that implements the TLObject interface using the TL boolean
// constructor IDs.
type TLBool bool

// ConstructorID returns BoolTrueID for true and BoolFalseID for false.
func (v TLBool) ConstructorID() uint32 {
	if v {
		return BoolTrueID
	}
	return BoolFalseID
}

// Encode writes the TL boolean encoding to b.
func (v TLBool) Encode(b *bytes.Buffer) error {
	WriteBool(b, bool(v))
	return nil
}

type GenericVector struct {
	Items []TLObject
}

func (v *GenericVector) ConstructorID() uint32 { return vectorBareID }

func (v *GenericVector) Encode(b *bytes.Buffer) error {
	WriteInt(b, vectorBareID)
	WriteInt(b, uint32(len(v.Items)))
	for _, item := range v.Items {
		if err := EncodeTLObject(b, item); err != nil {
			return err
		}
	}
	return nil
}

func WriteVectorObject(b *bytes.Buffer, items []TLObject) error {
	WriteInt(b, vectorBareID)
	WriteInt(b, uint32(len(items)))
	for _, item := range items {
		if err := EncodeTLObject(b, item); err != nil {
			return err
		}
	}
	return nil
}

func ReadVectorObject(r io.Reader) ([]TLObject, error) {
	count := ReadInt(r)
	if err := checkVectorCount(count); err != nil {
		return nil, err
	}
	items := make([]TLObject, count)
	for i := range items {
		obj, err := ReadTLObject(r)
		if err != nil {
			return nil, err
		}
		items[i] = obj
	}
	return items, nil
}

func init() {
	Registry[BoolTrueID] = func(r io.Reader) (TLObject, error) {
		return TLBool(true), nil
	}
	Registry[BoolFalseID] = func(r io.Reader) (TLObject, error) {
		return TLBool(false), nil
	}
	Registry[vectorBareID] = func(r io.Reader) (TLObject, error) {
		count := ReadInt(r)
		if err := checkVectorCount(count); err != nil {
			return nil, err
		}
		items := make([]TLObject, count)
		for i := range items {
			obj, err := ReadTLObject(r)
			if err != nil {
				return nil, err
			}
			items[i] = obj
		}
		return &GenericVector{Items: items}, nil
	}
}
