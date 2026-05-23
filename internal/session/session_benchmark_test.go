package session

import (
	"bytes"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

var (
	benchmarkRawResult []byte
	benchmarkTLResult  tg.TLObject
)

func benchmarkSession(b *testing.B) *Session {
	b.Helper()
	dc := DataCenter{ID: 2}
	st := newTestStorage()
	st.SetAuthKey(makeAuthKey())
	s, err := NewSession(dc, st, "TestDevice", "1.0", "en", "en")
	if err != nil {
		b.Fatalf("NewSession() error: %v", err)
	}
	return s
}

func encodeBenchmarkTLObject(b *testing.B, obj tg.TLObject) []byte {
	b.Helper()
	var buf bytes.Buffer
	if err := tg.EncodeTLObject(&buf, obj); err != nil {
		b.Fatalf("encode %T: %v", obj, err)
	}
	return buf.Bytes()
}

func benchmarkRPCResultBody(b *testing.B, reqMsgID int64, result tg.TLObject) []byte {
	b.Helper()
	var body bytes.Buffer
	tg.WriteInt(&body, tg.RPCResultTypeID)
	tg.WriteLong(&body, reqMsgID)
	body.Write(encodeBenchmarkTLObject(b, result))
	return body.Bytes()
}

func BenchmarkSessionRPCResultRouting(b *testing.B) {
	const reqMsgID = int64(123456)
	const respMsgID = int64(789012)
	pong := &tg.Pong{MsgID: reqMsgID, PingID: 42}
	body := benchmarkRPCResultBody(b, reqMsgID, pong)

	b.Run("constructor_fast_path_raw_result", func(b *testing.B) {
		s := benchmarkSession(b)
		b.ReportAllocs()

		for b.Loop() {
			handle := s.pending.Register(reqMsgID, true)
			s.handleRawPacket(respMsgID, body)
			<-handle.Done()
			_, benchmarkRawResult, _ = handle.Result()
			s.pending.Cancel(reqMsgID)
		}
	})

	b.Run("constructor_fast_path_decoded_result", func(b *testing.B) {
		s := benchmarkSession(b)
		b.ReportAllocs()

		for b.Loop() {
			handle := s.pending.Register(reqMsgID, false)
			s.handleRawPacket(respMsgID, body)
			<-handle.Done()
			benchmarkTLResult, _, _ = handle.Result()
			s.pending.Cancel(reqMsgID)
		}
	})

	b.Run("full_TL_decode_rpc_result_wrapper", func(b *testing.B) {
		s := benchmarkSession(b)
		b.ReportAllocs()

		for b.Loop() {
			handle := s.pending.Register(reqMsgID, false)
			r := tg.NewReader(body)
			obj, err := tg.ReadTLObject(r)
			tg.ReleaseReader(r)
			if err != nil {
				b.Fatalf("ReadTLObject() error: %v", err)
			}
			s.processIncoming(&tg.MTProtoMessage{MsgID: respMsgID, Body: obj})
			<-handle.Done()
			benchmarkTLResult, _, _ = handle.Result()
			s.pending.Cancel(reqMsgID)
		}
	})
}
