package tg

import (
	"encoding/binary"
	"io"
	"math"
	"sync"
)

type Reader struct {
	b   []byte
	off int
}

var readerPool = sync.Pool{
	New: func() any { return &Reader{b: make([]byte, 0, 4096)} },
}

func NewReader(data []byte) *Reader {
	r := readerPool.Get().(*Reader)
	if cap(r.b) < len(data) {
		r.b = make([]byte, len(data))
	} else {
		r.b = r.b[:len(data)]
	}
	copy(r.b, data)
	r.off = 0
	return r
}

func ReleaseReader(r *Reader) {
	r.b = r.b[:0]
	r.off = 0
	readerPool.Put(r)
}

func (r *Reader) Len() int { return len(r.b) - r.off }

func (r *Reader) ReadUint32() (uint32, error) {
	if r.off+4 > len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint32(r.b[r.off:])
	r.off += 4
	return v, nil
}

func (r *Reader) ReadInt32() (int32, error) {
	n, err := r.ReadUint32()
	return int32(n), err
}

func (r *Reader) ReadInt64() (int64, error) {
	if r.off+8 > len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	v := int64(binary.LittleEndian.Uint64(r.b[r.off:]))
	r.off += 8
	return v, nil
}

func (r *Reader) ReadFloat64() (float64, error) {
	if r.off+8 > len(r.b) {
		return 0, io.ErrUnexpectedEOF
	}
	bits := binary.LittleEndian.Uint64(r.b[r.off:])
	r.off += 8
	return math.Float64frombits(bits), nil
}

func (r *Reader) ReadInt128() ([16]byte, error) {
	var v [16]byte
	if r.off+16 > len(r.b) {
		return v, io.ErrUnexpectedEOF
	}
	copy(v[:], r.b[r.off:])
	r.off += 16
	return v, nil
}

func (r *Reader) ReadInt256() ([32]byte, error) {
	var v [32]byte
	if r.off+32 > len(r.b) {
		return v, io.ErrUnexpectedEOF
	}
	copy(v[:], r.b[r.off:])
	r.off += 32
	return v, nil
}

func (r *Reader) ReadBool() (bool, error) {
	id, err := r.ReadUint32()
	if err != nil {
		return false, err
	}
	return id == BoolTrueID, nil
}

func (r *Reader) ReadRawBytes(n int) ([]byte, error) {
	if r.off+n > len(r.b) {
		return nil, io.ErrUnexpectedEOF
	}
	v := r.b[r.off : r.off+n]
	r.off += n
	return v, nil
}

func (r *Reader) ReadBytes() ([]byte, error) {
	if r.off >= len(r.b) {
		return nil, io.ErrUnexpectedEOF
	}
	first := r.b[r.off]
	r.off++
	length := int(first)
	headerLen := 1

	if length > 253 {
		if r.off+3 > len(r.b) {
			return nil, io.ErrUnexpectedEOF
		}
		length = int(r.b[r.off]) | int(r.b[r.off+1])<<8 | int(r.b[r.off+2])<<16
		headerLen = 4
		r.off += 3
	}

	var data []byte
	if length > 0 {
		if r.off+length > len(r.b) {
			return nil, io.ErrUnexpectedEOF
		}
		data = make([]byte, length)
		copy(data, r.b[r.off:])
		r.off += length
	}

	padding := (4 - (length+headerLen)%4) % 4
	r.off += padding
	return data, nil
}

func (r *Reader) ReadString() (string, error) {
	if r.off >= len(r.b) {
		return "", io.ErrUnexpectedEOF
	}
	first := r.b[r.off]
	r.off++
	length := int(first)
	headerLen := 1

	if length > 253 {
		if r.off+3 > len(r.b) {
			return "", io.ErrUnexpectedEOF
		}
		length = int(r.b[r.off]) | int(r.b[r.off+1])<<8 | int(r.b[r.off+2])<<16
		headerLen = 4
		r.off += 3
	}

	var s string
	if length > 0 {
		if r.off+length > len(r.b) {
			return "", io.ErrUnexpectedEOF
		}
		s = string(r.b[r.off : r.off+length])
		r.off += length
	}

	padding := (4 - (length+headerLen)%4) % 4
	r.off += padding
	return s, nil
}

func (r *Reader) ReadVectorInt() ([]int32, error) {
	hdr, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	_ = hdr
	count, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if err := checkVectorCount(count); err != nil {
		return nil, err
	}
	if int(count)*4 > r.Len() {
		return nil, io.ErrUnexpectedEOF
	}
	result := make([]int32, count)
	for i := range result {
		v, err := r.ReadInt32()
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

func (r *Reader) ReadVectorLong() ([]int64, error) {
	hdr, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	_ = hdr
	count, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if err := checkVectorCount(count); err != nil {
		return nil, err
	}
	if int(count)*8 > r.Len() {
		return nil, io.ErrUnexpectedEOF
	}
	result := make([]int64, count)
	for i := range result {
		v, err := r.ReadInt64()
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

func (r *Reader) ReadVectorString() ([]string, error) {
	hdr, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	_ = hdr
	count, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if err := checkVectorCount(count); err != nil {
		return nil, err
	}
	result := make([]string, count)
	for i := range result {
		v, err := r.ReadString()
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

func (r *Reader) ReadVectorBytes() ([][]byte, error) {
	hdr, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	_ = hdr
	count, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if err := checkVectorCount(count); err != nil {
		return nil, err
	}
	result := make([][]byte, count)
	for i := range result {
		v, err := r.ReadBytes()
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}
