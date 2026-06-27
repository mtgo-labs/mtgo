package tg

import (
	"bytes"
	"testing"
)

// --- TL encode benchmarks for common types ---

func BenchmarkEncodePong(b *testing.B) {
	v := &Pong{MsgID: 123456, PingID: 42}
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		v.Encode(&buf)
	}
}

func BenchmarkEncodeConfig(b *testing.B) {
	v := &Config{
		Date:              1700000000,
		Expires:           1700003600,
		ThisDC:            2,
		ChatSizeMax:       200,
		MegagroupSizeMax:  200000,
		ForwardedCountMax: 100,
		EditTimeLimit:     172800,
		MessageLengthMax:  4096,
		CaptionLengthMax:  1024,
		WebfileDCID:       4,
		MeURLPrefix:       "https://t.me/",
		DCTxtDomainName:   "v4.web.telegram.org",
	}
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		v.Encode(&buf)
	}
}

func BenchmarkDecodeConfig(b *testing.B) {
	v := &Config{
		Date:             1700000000,
		Expires:          1700003600,
		ThisDC:           2,
		ChatSizeMax:      200,
		MeURLPrefix:      "https://t.me/",
		DCTxtDomainName:  "v4.web.telegram.org",
		EditTimeLimit:    172800,
		MessageLengthMax: 4096,
	}
	var buf bytes.Buffer
	EncodeTLObject(&buf, v) // includes constructor ID
	encoded := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r := NewReader(encoded)
		_, err := ReadTLObject(r)
		ReleaseReader(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeMsgsAck(b *testing.B) {
	v := &MsgsAck{MsgIds: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		v.Encode(&buf)
	}
}

func BenchmarkEncodeMTProtoMessage(b *testing.B) {
	v := &MTProtoMessage{
		MsgID: 1234567890,
		SeqNo: 1,
		Body:  &Pong{MsgID: 1234567890, PingID: 42},
	}
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		v.Encode(&buf)
	}
}

func BenchmarkDecodeMTProtoMessage(b *testing.B) {
	v := &MTProtoMessage{
		MsgID: 1234567890,
		SeqNo: 1,
		Body:  &Pong{MsgID: 1234567890, PingID: 42},
	}
	var buf bytes.Buffer
	v.Encode(&buf)
	encoded := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r := NewReader(encoded)
		_, err := DecodeMTProtoMessage(r)
		ReleaseReader(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Gzip packed ---

func BenchmarkGzipPackedEncode(b *testing.B) {
	payload := make([]byte, 512)
	for i := range payload {
		payload[i] = byte(i)
	}
	inner := &Pong{MsgID: 123456, PingID: 42}
	var innerBuf bytes.Buffer
	inner.Encode(&innerBuf)

	v := &GzipPacked{Data: inner}
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		v.Encode(&buf)
	}
}

// --- Reader pool ---

func BenchmarkNewReaderRelease(b *testing.B) {
	data := make([]byte, 128)
	b.ReportAllocs()
	for b.Loop() {
		r := NewReader(data)
		ReleaseReader(r)
	}
}

// --- EncodeTLObject (generic dispatch) ---

func BenchmarkEncodeTLObject(b *testing.B) {
	v := &Pong{MsgID: 123456, PingID: 42}
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		EncodeTLObject(&buf, v)
	}
}

func BenchmarkReadTLObject(b *testing.B) {
	v := &Pong{MsgID: 123456, PingID: 42}
	var buf bytes.Buffer
	v.Encode(&buf)
	encoded := buf.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r := NewReader(encoded)
		_, err := ReadTLObject(r)
		ReleaseReader(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}
