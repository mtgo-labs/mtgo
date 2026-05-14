package tg

import (
	"bytes"
	"compress/gzip"
	"io"
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
	var buf bytes.Buffer
	if err := EncodeTLObject(&buf, g.Data); err != nil {
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
	var buf bytes.Buffer
	if _, err := buf.Write(data.Raw); err != nil {
		return nil, err
	}
	return ReadTLObject(&buf)
}

func DecodeGzipPacked(r io.Reader) (*GzipPacked, error) {
	compressed := ReadString(r)
	decompressed, err := gzipDecompress([]byte(compressed))
	if err != nil {
		return nil, err
	}
	return &GzipPacked{Data: &GzipPackedData{Raw: decompressed}}, nil
}

type GzipPackedData struct {
	Raw []byte
}

func (d *GzipPackedData) Encode(b *bytes.Buffer) error {
	_, err := b.Write(d.Raw)
	return err
}

func (d *GzipPackedData) ConstructorID() uint32 { return 0 }

func gzipCompress(data []byte) string {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.String()
}

func gzipDecompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func init() {
	Registry[GzipPackedID] = func(r io.Reader) (TLObject, error) {
		return DecodeGzipPacked(r)
	}
}
