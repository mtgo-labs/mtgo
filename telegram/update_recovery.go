package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

type differenceRPC interface {
	UpdatesGetDifference(ctx context.Context, req *tg.UpdatesGetDifferenceRequest) (tg.DifferenceClass, error)
	UpdatesGetChannelDifference(ctx context.Context, req *tg.UpdatesGetChannelDifferenceRequest) (tg.ChannelDifferenceClass, error)
}

func (m *updateManager) RecoverAccount(ctx context.Context, rpc differenceRPC) error {
	m.mu.Lock()
	if m.recovering {
		m.mu.Unlock()
		return nil
	}
	m.recovering = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.recovering = false
		m.mu.Unlock()
	}()

	for {
		m.mu.Lock()
		req := &tg.UpdatesGetDifferenceRequest{
			PTS:  m.state.Pts,
			Date: m.state.Date,
			Qts:  m.state.Qts,
		}
		m.mu.Unlock()
		diff, err := rpc.UpdatesGetDifference(ctx, req)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDifferenceRecovery, err)
		}
		done, err := m.applyDifference(ctx, diff)
		if err != nil {
			return err
		}
		if done {
			m.mu.Lock()
			m.health.LastRecovery = time.Now()
			m.health.RecoveryCount++
			m.mu.Unlock()
			return nil
		}
	}
}

func (m *updateManager) applyDifference(ctx context.Context, diff tg.DifferenceClass) (done bool, err error) {
	switch d := diff.(type) {
	case *tg.UpdatesDifferenceEmpty:
		m.mu.Lock()
		m.state.Date = d.Date
		m.state.Seq = d.Seq
		m.mu.Unlock()
		m.advanceState(updateMeta{Date: d.Date, Seq: d.Seq})
		return true, nil

	case *tg.UpdatesDifference:
		m.applyDifferenceUpdates(ctx, d.NewMessages, d.OtherUpdates)
		if d.State != nil {
			m.mu.Lock()
			m.state.Pts = d.State.PTS
			m.state.Qts = d.State.Qts
			m.state.Date = d.State.Date
			m.state.Seq = d.State.Seq
			m.mu.Unlock()
			m.advanceState(updateMeta{
				Pts:  d.State.PTS,
				Qts:  d.State.Qts,
				Date: d.State.Date,
				Seq:  d.State.Seq,
			})
		}
		return true, nil

	case *tg.UpdatesDifferenceSlice:
		m.applyDifferenceUpdates(ctx, d.NewMessages, d.OtherUpdates)
		if d.IntermediateState != nil {
			m.mu.Lock()
			m.state.Pts = d.IntermediateState.PTS
			m.state.Qts = d.IntermediateState.Qts
			m.state.Date = d.IntermediateState.Date
			m.state.Seq = d.IntermediateState.Seq
			m.mu.Unlock()
			m.advanceState(updateMeta{
				Pts:  d.IntermediateState.PTS,
				Qts:  d.IntermediateState.Qts,
				Date: d.IntermediateState.Date,
				Seq:  d.IntermediateState.Seq,
			})
		}
		return false, nil

	case *tg.UpdatesDifferenceTooLong:
		m.mu.Lock()
		m.state.Pts = d.PTS
		m.mu.Unlock()
		m.advanceState(updateMeta{Pts: d.PTS})
		return false, nil

	default:
		return true, fmt.Errorf("%w: unknown difference type %T", ErrDifferenceRecovery, diff)
	}
}

func (m *updateManager) applyDifferenceUpdates(ctx context.Context, messages []tg.MessageClass, updates []tg.UpdateClass) {
	for _, msg := range messages {
		upd := &tg.UpdateNewMessage{
			Message:  msg,
			PTS:      0,
			PTSCount: 0,
		}
		if err := m.applyUpdate(ctx, upd, nil, nil, nil, nil); err != nil {
			m.client.Log.Warnf("apply difference update: %v", err)
		}
	}
	for _, upd := range updates {
		if err := m.applyUpdate(ctx, upd, nil, nil, nil, nil); err != nil {
			m.client.Log.Warnf("apply difference update: %v", err)
		}
	}
}

func (m *updateManager) RecoverChannel(ctx context.Context, rpc differenceRPC, channelID int64, input tg.InputChannelClass) error {
	m.mu.Lock()
	ch, ok := m.channels[channelID]
	if !ok {
		ch = channelState{ChannelID: channelID}
	}
	if ch.recovering {
		m.mu.Unlock()
		return nil
	}
	ch.recovering = true
	m.channels[channelID] = ch
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		if c, exists := m.channels[channelID]; exists {
			c.recovering = false
			m.channels[channelID] = c
		}
		m.mu.Unlock()
	}()

	for {
		m.mu.Lock()
		currentPts := m.channels[channelID].Pts
		m.mu.Unlock()

		req := &tg.UpdatesGetChannelDifferenceRequest{
			Channel: input,
			Filter:  &tg.ChannelMessagesFilterEmpty{},
			PTS:     currentPts,
			Limit:   100,
		}
		result, err := rpc.UpdatesGetChannelDifference(ctx, req)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrChannelDifference, err)
		}

		done, err := m.applyChannelDifference(ctx, result, channelID)
		if err != nil {
			return err
		}
		if done {
			m.mu.Lock()
			m.health.LastRecovery = time.Now()
			m.health.RecoveryCount++
			m.mu.Unlock()
			return nil
		}
	}
}

func (m *updateManager) applyChannelDifference(ctx context.Context, result tg.ChannelDifferenceClass, channelID int64) (bool, error) {
	switch d := result.(type) {
	case *tg.UpdatesChannelDifferenceEmpty:
		m.advanceState(updateMeta{IsChannel: true, ChannelID: channelID, ChannelPts: d.PTS})
		return d.Final, nil

	case *tg.UpdatesChannelDifference:
		m.applyChannelDifferenceUpdates(ctx, d.NewMessages, d.OtherUpdates)
		m.advanceState(updateMeta{IsChannel: true, ChannelID: channelID, ChannelPts: d.PTS})
		return d.Final, nil

	case *tg.UpdatesChannelDifferenceTooLong:
		if dialog, ok := d.Dialog.(*tg.Dialog); ok && dialog.PTS != 0 {
			m.advanceState(updateMeta{IsChannel: true, ChannelID: channelID, ChannelPts: dialog.PTS})
		}
		return d.Final, nil

	default:
		return true, fmt.Errorf("%w: unknown channel difference type %T", ErrChannelDifference, result)
	}
}

func (m *updateManager) applyChannelDifferenceUpdates(ctx context.Context, messages []tg.MessageClass, updates []tg.UpdateClass) {
	for _, msg := range messages {
		upd := &tg.UpdateNewChannelMessage{
			Message:  msg,
			PTS:      0,
			PTSCount: 0,
		}
		if err := m.applyUpdate(ctx, upd, nil, nil, nil, nil); err != nil {
			m.client.Log.Warnf("apply channel difference update: %v", err)
		}
	}
	for _, upd := range updates {
		if err := m.applyUpdate(ctx, upd, nil, nil, nil, nil); err != nil {
			m.client.Log.Warnf("apply channel difference update: %v", err)
		}
	}
}
