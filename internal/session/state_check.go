package session

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

const (
	// stateCheckInterval is how often the loop scans for overdue pending RPCs.
	stateCheckInterval = 500 * time.Millisecond
	// stateCheckThreshold is the minimum age before a pending RPC is considered
	// overdue and eligible for msgs_state_req probing.
	stateCheckThreshold = 1500 * time.Millisecond
	// stateReqTimeout is how long to wait for a msgs_state_info response before
	// treating all queried messages as potentially lost.
	stateReqTimeout = 2500 * time.Millisecond
)

// pendingStateReq tracks a sent msgs_state_req and the msg_ids it queried.
type pendingStateReq struct {
	msgIDs []int64 // original pending msg_ids we asked about
	sentAt time.Time
}

// stateCheckLoop is an errgroup goroutine that periodically sends msgs_state_req
// for content-related pending RPCs that have been sent more than
// stateCheckThreshold ago without receiving an ack or result. When the server
// responds with msgs_state_info, the status bytes determine whether the
// message was received, lost, or its response was generated but not delivered.
//
// This implements the MTProto service message reconciliation described at
// https://core.telegram.org/mtproto/service_messages_about_messages
// Ported from gotd/td rpc/engine.go retryUntilAck and mtcute
// session-connection.ts getStateSchedule/GET_STATE_INTERVAL.
func (s *Session) stateCheckLoop(ctx context.Context) error {
	ticker := time.NewTicker(stateCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		s.checkPendingStates()
		s.expireStateReqs()
	}
}

// checkPendingStates collects overdue pending RPCs and sends msgs_state_req.
func (s *Session) checkPendingStates() {
	ids := s.pending.OverduePending(stateCheckThreshold)
	if len(ids) == 0 {
		return
	}

	stateReqMsgID := s.sendStateReq(ids)
	if stateReqMsgID == 0 {
		return
	}

	s.stateReqMu.Lock()
	s.stateReqs[stateReqMsgID] = &pendingStateReq{
		msgIDs: ids,
		sentAt: time.Now(),
	}
	s.stateReqMu.Unlock()

	if s.log != nil {
		s.log.Debugf("state-check: sent msgs_state_req for %d pending queries msg_id=%d", len(ids), stateReqMsgID)
	}
}

// sendStateReq builds and sends a msgs_state_req service message, returning
// the msg_id of the sent request (0 on failure).
func (s *Session) sendStateReq(msgIDs []int64) int64 {
	select {
	case <-s.done:
		return 0
	default:
	}

	s.mu.RLock()
	authKey := s.authKey
	authKeyID := s.authKeyID
	s.mu.RUnlock()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(false)
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: uint32(seqNo),
		Body:  &tg.MsgsStateReq{MsgIds: msgIDs},
	}
	encrypted, err := crypto.Pack(message, s.saltMgr.Load(), s.sessionIDBytes(), authKey, authKeyID)
	if err != nil {
		return 0
	}
	if err := s.writeEncryptedDirect(encrypted, 10*time.Second); err != nil {
		if s.log != nil {
			s.log.Warnf("state-check: failed to send msgs_state_req: %v", err)
		}
		return 0
	}
	return msgID
}

// handleStateInfo processes a msgs_state_info response, interpreting the status
// bytes for each queried message.
func (s *Session) handleStateInfo(reqMsgID int64, info string) {
	s.stateReqMu.Lock()
	req, ok := s.stateReqs[reqMsgID]
	if ok {
		delete(s.stateReqs, reqMsgID)
	}
	s.stateReqMu.Unlock()

	if !ok {
		// Late or duplicate response for a request we already expired or never sent.
		return
	}

	for i, msgID := range req.msgIDs {
		if i >= len(info) {
			break
		}
		status := info[i]
		s.interpretStateByte(msgID, status)
	}
}

// interpretStateByte acts on a single MTProto status byte for the given msgID.
//
// Status byte layout (bits 0-2 = state):
//
//	1 = unknown — keep waiting.
//	2 = not received.
//	3 = not received because msg_id is too high or too low.
//	4 = received.
//
// bit 3 (0x08): acknowledged. bit 4 (0x10): acknowledgement not required.
// bit 5 (0x20): being processed. bit 6 (0x40): response generated.
// bit 7 (0x80): other related message sent.
func (s *Session) interpretStateByte(msgID int64, status byte) {
	switch status & 0x07 {
	case 2, 3:
		// Message was not received by the server. Reject the pending handle so
		// the caller's retry loop can re-send.
		if s.pending.Reject(msgID, ErrMsgNotReceived) {
			if s.log != nil {
				s.log.Warnf("state-check: msg_id=%d not received by server (status=0x%02x), rejecting", msgID, status)
			}
		}
	case 4:
		// Receipt is confirmed even when the response is still being processed.
		// Marking the handle acknowledged removes it from future state probes.
		s.pending.MarkAcked(msgID)
	default:
		// Unknown or malformed state. Keep waiting rather than risk replaying a
		// request whose delivery cannot be disproved.
	}
}

// expireStateReqs cleans up state requests that have timed out without
// receiving a msgs_state_info response. Timed-out queries are NOT rejected —
// the caller's context deadline is the ultimate backstop, and a missing
// state_info response could mean the server didn't understand the request.
func (s *Session) expireStateReqs() {
	cutoff := time.Now().Add(-stateReqTimeout)

	s.stateReqMu.Lock()
	for msgID, req := range s.stateReqs {
		if req.sentAt.Before(cutoff) {
			delete(s.stateReqs, msgID)
		}
	}
	s.stateReqMu.Unlock()
}

// handleRawMsgsStateInfo parses a raw msgs_state_info body (after the
// constructor ID) and dispatches to handleStateInfo.
func (s *Session) handleRawMsgsStateInfo(body []byte) {
	if len(body) < 8 {
		return
	}
	reqMsgID := int64(binary.LittleEndian.Uint64(body[:8]))
	r := tg.NewReader(body[8:])
	defer tg.ReleaseReader(r)
	info, err := r.ReadString()
	if err != nil {
		return
	}
	s.handleStateInfo(reqMsgID, info)
}
