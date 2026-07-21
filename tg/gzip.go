package tg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/klauspost/compress/gzip"
)

// GzipPackedID is the TL constructor ID for the gzip_packed type.
const GzipPackedID = 0x3072CFA1

// GzipPacked represents a gzip-compressed TL object.
type GzipPacked struct {
	// Data holds the inner TLObject, typically a GzipPackedData after decoding.
	Data TLObject
}

// ConstructorID returns the TL constructor ID for GzipPacked.
func (g *GzipPacked) ConstructorID() uint32 { return GzipPackedID }

// Encode compresses the inner TLObject and writes it to b in TL format.
func (g *GzipPacked) Encode(b *bytes.Buffer) error {
	buf := gzipBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer gzipBufPool.Put(buf)

	if err := EncodeTLObject(buf, g.Data); err != nil {
		return err
	}
	WriteString(b, gzipCompress(buf.Bytes()))
	return nil
}

func (g *GzipPacked) Decode() (TLObject, error) {
	data, ok := g.Data.(*GzipPackedData)
	if !ok {
		return nil, &UnknownConstructorError{ID: GzipPackedID}
	}
	r := NewReader(data.Raw)
	defer ReleaseReader(r)
	return ReadTLObject(r)
}

func DecodeGzipPacked(r *Reader) (*GzipPacked, error) {
	compressed, err := r.ReadString()
	if err != nil {
		return nil, err
	}
	decompressed, err := gzipDecompress([]byte(compressed))
	if err != nil {
		return nil, err
	}
	return &GzipPacked{Data: &GzipPackedData{Raw: decompressed}}, nil
}

// PeekGzipPackedConstructor returns the constructor ID at the start of a
// TL-encoded gzip_packed bytes field without copying or fully decompressing it.
func PeekGzipPackedConstructor(data []byte) (uint32, error) {
	compressed, err := readTLBytesView(data)
	if err != nil {
		return 0, err
	}

	zr := gzipReaderPool.Get().(*gzip.Reader)
	defer gzipReaderPool.Put(zr)
	if err := zr.Reset(bytes.NewReader(compressed)); err != nil {
		return 0, err
	}

	var constructor [4]byte
	if _, err := io.ReadFull(zr, constructor[:]); err != nil {
		return 0, fmt.Errorf("gzip: read constructor: %w", err)
	}
	return binary.LittleEndian.Uint32(constructor[:]), nil
}

func readTLBytesView(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	headerLen := 1
	length := int(data[0])
	if length > 253 {
		if len(data) < 4 {
			return nil, io.ErrUnexpectedEOF
		}
		headerLen = 4
		length = int(data[1]) | int(data[2])<<8 | int(data[3])<<16
	}

	end := headerLen + length
	padding := (4 - (length+headerLen)%4) % 4
	if end+padding > len(data) {
		return nil, io.ErrUnexpectedEOF
	}
	return data[headerLen:end], nil
}

type GzipPackedData struct {
	Raw []byte
}

func (d *GzipPackedData) Encode(b *bytes.Buffer) error {
	_, err := b.Write(d.Raw)
	return err
}

func (d *GzipPackedData) ConstructorID() uint32 { return 0 }

var (
	gzipReaderPool = sync.Pool{
		New: func() any { return new(gzip.Reader) },
	}
	gzipWriterPool = sync.Pool{
		New: func() any { return gzip.NewWriter(nil) },
	}
	gzipBufPool = sync.Pool{
		New: func() any { return bytes.NewBuffer(make([]byte, 0, 4096)) },
	}
)

var errDecompressionBomb = errors.New("gzip: decompression bomb detected")

func gzipCompress(data []byte) string {
	buf := gzipBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer gzipBufPool.Put(buf)

	w := gzipWriterPool.Get().(*gzip.Writer)
	w.Reset(buf)
	defer gzipWriterPool.Put(w)

	w.Write(data)
	w.Close()
	return buf.String()
}

func gzipDecompress(data []byte) ([]byte, error) {
	r := gzipReaderPool.Get().(*gzip.Reader)
	defer gzipReaderPool.Put(r)

	if err := r.Reset(bytes.NewReader(data)); err != nil {
		return nil, err
	}

	buf := gzipBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer gzipBufPool.Put(buf)

	// Decompression bomb protection: limit to 10 MB like gotd/td.
	const maxUncompressed = 10 * 1024 * 1024
	lr := io.LimitReader(r, maxUncompressed+1)
	n, err := buf.ReadFrom(lr)
	if err != nil {
		return nil, fmt.Errorf("gzip: decompress: %w", err)
	}
	if n > maxUncompressed {
		return nil, errDecompressionBomb
	}

	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

func init() {
	Registry[GzipPackedID] = func(r *Reader) (TLObject, error) {
		return DecodeGzipPacked(r)
	}
}
