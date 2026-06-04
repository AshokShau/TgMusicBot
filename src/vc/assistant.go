package vc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"ashokshau/tgmusic/src/vc/ntgcalls"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const connectWaitTimeout = 25 * time.Second

type pendingConnection struct {
	MediaDescription ntgcalls.MediaDescription
	Payload          string
}

type Assistant struct {
	App     *tg.Client
	binding *ntgcalls.Client
	self    *tg.UserObj

	mu                 sync.RWMutex
	mutedByAdmin       []int64
	presentations      []int64
	inputGroupCalls    map[int64]tg.InputGroupCall
	pendingConnections map[int64]*pendingConnection
	waitConnect        map[int64]chan error

	streamEndCallbacks []ntgcalls.StreamEndCallback
}

func newAssistant(app *tg.Client) (*Assistant, error) {
	a := &Assistant{
		App:                app,
		binding:            ntgcalls.NTgCalls(),
		inputGroupCalls:    make(map[int64]tg.InputGroupCall),
		pendingConnections: make(map[int64]*pendingConnection),
		waitConnect:        make(map[int64]chan error),
	}
	if app.IsConnected() {
		self, err := app.GetMe()
		if err != nil {
			return nil, fmt.Errorf("failed to get self user: %w", err)
		}
		a.self = self
	}
	a.handleUpdates()
	return a, nil
}

func (a *Assistant) OnStreamEnd(callback ntgcalls.StreamEndCallback) {
	a.streamEndCallbacks = append(a.streamEndCallbacks, callback)
}

func (a *Assistant) Close() {
	a.binding.Free()
}

func (a *Assistant) Play(ctx context.Context, chatId int64, mediaDescription ntgcalls.MediaDescription) error {
	if a.binding.Calls()[chatId] != nil {
		return a.binding.SetStreamSources(chatId, ntgcalls.CaptureStream, mediaDescription)
	}
	if err := a.connectCall(ctx, chatId, mediaDescription, ""); err != nil {
		return err
	}
	if chatId < 0 {
		return a.joinPresentation(ctx, chatId, mediaDescription.Screen != nil)
	}
	return nil
}

func (a *Assistant) stopCall(ctx context.Context, chatId int64) error {
	a.mu.Lock()
	a.presentations = stdRemove(a.presentations, chatId)
	delete(a.pendingConnections, chatId)
	inputGroupCall := a.inputGroupCalls[chatId]
	a.mu.Unlock()

	if err := a.binding.Stop(chatId); err != nil {
		return err
	}

	if inputGroupCall == nil {
		return nil
	}
	_, err := a.App.PhoneLeaveGroupCall(inputGroupCall, 0)
	return err
}

func (a *Assistant) connectCall(ctx context.Context, chatId int64, mediaDescription ntgcalls.MediaDescription, jsonParams string) error {
	connectCh := make(chan error, 1)
	a.mu.Lock()
	a.waitConnect[chatId] = connectCh
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.waitConnect, chatId)
		a.mu.Unlock()
	}()

	if chatId < 0 {
		if a.self == nil {
			return errors.New("assistant is not ready")
		}
		var err error
		jsonParams, err = a.binding.CreateCall(chatId)
		if err != nil {
			_ = a.binding.Stop(chatId)
			return err
		}

		if err = a.binding.SetStreamSources(chatId, ntgcalls.CaptureStream, mediaDescription); err != nil {
			_ = a.binding.Stop(chatId)
			return err
		}

		inputGroupCall, err := a.getInputGroupCall(ctx, chatId)
		if err != nil {
			_ = a.binding.Stop(chatId)
			return err
		}

		resultParams := "{\"transport\": null}"
		callResRaw, err := a.App.PhoneJoinGroupCall(
			&tg.PhoneJoinGroupCallParams{
				Muted:        false,
				VideoStopped: mediaDescription.Camera == nil,
				Call:         inputGroupCall,
				Params: &tg.DataJson{
					Data: jsonParams,
				},
				JoinAs: &tg.InputPeerUser{
					UserID:     a.self.ID,
					AccessHash: a.self.AccessHash,
				},
			},
		)
		if err != nil {
			return err
		}

		callRes := callResRaw.(*tg.UpdatesObj)
		for _, update := range callRes.Updates {
			if connUpdate, ok := update.(*tg.UpdateGroupCallConnection); ok {
				resultParams = connUpdate.Params.Data
			}
		}

		if err = a.binding.Connect(
			chatId,
			resultParams,
			false,
		); err != nil {
			return err
		}

		connectionMode, err := a.binding.GetConnectionMode(chatId)
		if err != nil {
			return err
		}

		if connectionMode == ntgcalls.StreamConnection && len(jsonParams) > 0 {
			a.mu.Lock()
			a.pendingConnections[chatId] = &pendingConnection{
				MediaDescription: mediaDescription,
				Payload:          jsonParams,
			}
			a.mu.Unlock()
		}
	} else {
		return errors.New("p2p is not supported")
	}

	select {
	case err := <-connectCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(connectWaitTimeout):
		return fmt.Errorf("connection timeout")
	}
}

func (a *Assistant) joinPresentation(ctx context.Context, chatId int64, join bool) error {
	connectionMode, err := a.binding.GetConnectionMode(chatId)
	if err != nil {
		return err
	}
	if connectionMode == ntgcalls.StreamConnection {
		return nil
	}
	if connectionMode != ntgcalls.RtcConnection {
		return nil
	}

	if join {
		a.mu.RLock()
		already := slices.Contains(a.presentations, chatId)
		a.mu.RUnlock()
		if already {
			return nil
		}

		connectCh := make(chan error, 1)
		a.mu.Lock()
		a.waitConnect[chatId] = connectCh
		a.mu.Unlock()

		defer func() {
			a.mu.Lock()
			delete(a.waitConnect, chatId)
			a.mu.Unlock()
		}()

		jsonParams, err := a.binding.InitPresentation(chatId)
		if err != nil {
			return err
		}
		resultParams := "{\"transport\": null}"
		inputGroupCall, err := a.getInputGroupCall(ctx, chatId)
		if err != nil {
			return err
		}
		callResRaw, err := a.App.PhoneJoinGroupCallPresentation(
			inputGroupCall,
			&tg.DataJson{
				Data: jsonParams,
			},
		)
		if err != nil {
			return err
		}
		callRes := callResRaw.(*tg.UpdatesObj)
		for _, update := range callRes.Updates {
			if connUpdate, ok := update.(*tg.UpdateGroupCallConnection); ok {
				resultParams = connUpdate.Params.Data
			}
		}
		if err = a.binding.Connect(chatId, resultParams, true); err != nil {
			return err
		}

		select {
		case err := <-connectCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(connectWaitTimeout):
			return fmt.Errorf("presentation connection timeout")
		}

		a.mu.Lock()
		a.presentations = append(a.presentations, chatId)
		a.mu.Unlock()
		return nil
	}

	a.mu.RLock()
	inPresentation := slices.Contains(a.presentations, chatId)
	a.mu.RUnlock()
	if inPresentation {
		a.mu.Lock()
		a.presentations = stdRemove(a.presentations, chatId)
		a.mu.Unlock()
		if err = a.binding.StopPresentation(chatId); err != nil {
			return err
		}
		a.mu.RLock()
		inputGroupCall := a.inputGroupCalls[chatId]
		a.mu.RUnlock()
		if inputGroupCall != nil {
			_, err = a.App.PhoneLeaveGroupCallPresentation(inputGroupCall)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Assistant) getInputGroupCall(ctx context.Context, chatId int64) (tg.InputGroupCall, error) {
	a.mu.RLock()
	call, ok := a.inputGroupCalls[chatId]
	a.mu.RUnlock()
	if ok {
		if call == nil {
			return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
		}
		return call, nil
	}

	peer, err := a.App.ResolvePeer(chatId)
	if err != nil {
		return nil, err
	}
	switch chatPeer := peer.(type) {
	case *tg.InputPeerChannel:
		fullChat, err := a.App.ChannelsGetFullChannel(
			&tg.InputChannelObj{
				ChannelID:  chatPeer.ChannelID,
				AccessHash: chatPeer.AccessHash,
			},
		)
		if err != nil {
			return nil, err
		}
		a.mu.Lock()
		a.inputGroupCalls[chatId] = fullChat.FullChat.(*tg.ChannelFull).Call
		a.mu.Unlock()
	case *tg.InputPeerChat:
		fullChat, err := a.App.MessagesGetFullChat(chatPeer.ChatID)
		if err != nil {
			return nil, err
		}
		a.mu.Lock()
		a.inputGroupCalls[chatId] = fullChat.FullChat.(*tg.ChatFullObj).Call
		a.mu.Unlock()
	default:
		return nil, fmt.Errorf("chatId %d is not a group call", chatId)
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	if call, ok := a.inputGroupCalls[chatId]; ok && call == nil {
		return nil, fmt.Errorf("group call for chatId %d is closed", chatId)
	} else if ok {
		return call, nil
	}

	return nil, fmt.Errorf("group call for chatId %d not found", chatId)
}

func (a *Assistant) setCallStatus(ctx context.Context, call tg.InputGroupCall, state ntgcalls.MediaState) error {
	if call == nil {
		return errors.New("missing input group call")
	}
	if a.self == nil {
		return errors.New("assistant is not ready")
	}
	_, err := a.App.PhoneEditGroupCallParticipant(
		&tg.PhoneEditGroupCallParticipantParams{
			Call: call,
			Participant: &tg.InputPeerUser{
				UserID:     a.self.ID,
				AccessHash: a.self.AccessHash,
			},
			Muted:              state.Muted,
			VideoPaused:        state.VideoPaused,
			VideoStopped:       state.VideoStopped,
			PresentationPaused: state.PresentationPaused,
		},
	)
	return err
}

func (a *Assistant) parseChatId(ctx context.Context, chatId any) (int64, error) {
	if chatId == nil {
		return 0, fmt.Errorf("chatId cannot be nil")
	}

	var parsedChatId int64
	switch v := chatId.(type) {
	case tg.Peer:
		switch p := v.(type) {
		case *tg.PeerUser:
			parsedChatId = p.UserID
		case *tg.PeerChat:
			parsedChatId = -p.ChatID
		case *tg.PeerChannel:
			parsedChatId = -1000000000000 - p.ChannelID
		}
	case int64:
		parsedChatId = v
	case int:
		parsedChatId = int64(v)
	case int32:
		parsedChatId = int64(v)
	case int16:
		parsedChatId = int64(v)
	case int8:
		parsedChatId = int64(v)
	case string:
		rawChat, err := a.App.ResolveUsername(v)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve username: %w", err)
		}
		switch chat := rawChat.(type) {
		case *tg.UserObj:
			parsedChatId = chat.ID
		case *tg.ChatObj:
			parsedChatId = -chat.ID
		case *tg.Channel:
			parsedChatId = -1000000000000 - chat.ID
		}
	default:
		return 0, fmt.Errorf("unsupported chatId type: %T", chatId)
	}

	switch chatId.(type) {
	case int64, int, int32, int16, int8:
		rawChat, err := a.App.GetInputPeer(parsedChatId)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve peer: %w", err)
		}
		switch peer := rawChat.(type) {
		case *tg.InputPeerUser:
			parsedChatId = peer.UserID
		case *tg.InputPeerChat:
			parsedChatId = -peer.ChatID
		case *tg.InputPeerChannel:
			parsedChatId = -1000000000000 - peer.ChannelID
		}
	}
	return parsedChatId, nil
}

func (a *Assistant) handleUpdates() {
	a.App.AddRawHandler(&tg.UpdateGroupCallParticipants{}, a.onGroupCallParticipants)
	a.App.AddRawHandler(&tg.UpdateGroupCall{}, a.onGroupCall)

	a.binding.OnRequestBroadcastTimestamp(a.onRequestBroadcastTimestamp)
	a.binding.OnRequestBroadcastPart(a.onRequestBroadcastPart)
	a.binding.OnConnectionChange(a.onConnectionChange)
	a.binding.OnUpgrade(a.onUpgrade)

	a.binding.OnStreamEnd(func(chatId int64, streamType ntgcalls.StreamType, streamDevice ntgcalls.StreamDevice) {
		for _, callback := range a.streamEndCallbacks {
			go callback(chatId, streamType, streamDevice)
		}
	})
}

func (a *Assistant) onGroupCallParticipants(m tg.Update, _ *tg.Client) error {
	participantsUpdate := m.(*tg.UpdateGroupCallParticipants)
	inputCall, ok := participantsUpdate.Call.(*tg.InputGroupCallObj)
	if !ok {
		return nil
	}
	chatId, err := a.convertGroupCallId(inputCall.ID)
	if err != nil {
		return nil
	}
	if a.self == nil {
		return nil
	}

	for _, participant := range participantsUpdate.Participants {
		participantId := getParticipantId(participant.Peer)
		if participantId != a.self.ID {
			continue
		}

		connectionMode, err := a.binding.GetConnectionMode(chatId)
		if err == nil && connectionMode == ntgcalls.StreamConnection && participant.CanSelfUnmute {
			a.mu.RLock()
			pending := a.pendingConnections[chatId]
			a.mu.RUnlock()
			if pending != nil {
				err = a.connectCall(
					context.Background(),
					chatId,
					pending.MediaDescription,
					pending.Payload,
				)
				if err != nil {
					a.App.Log.Warnf("failed to reconnect pending_call: %v", err)
				}
				a.mu.Lock()
				delete(a.pendingConnections, chatId)
				a.mu.Unlock()
			}
			break
		}

		if !participant.CanSelfUnmute {
			a.mu.Lock()
			if !slices.Contains(a.mutedByAdmin, chatId) {
				a.mutedByAdmin = append(a.mutedByAdmin, chatId)
			}
			a.mu.Unlock()
			break
		}

		a.mu.RLock()
		wasMuted := slices.Contains(a.mutedByAdmin, chatId)
		a.mu.RUnlock()
		if wasMuted {
			state, stateErr := a.binding.GetState(chatId)
			if stateErr != nil {
				a.App.Log.Warnf("failed to get call state: %v", stateErr)
				break
			}
			if statusErr := a.setCallStatus(context.Background(), participantsUpdate.Call, state); statusErr != nil {
				a.App.Log.Warnf("failed to update call status: %v", statusErr)
				break
			}
			a.mu.Lock()
			a.mutedByAdmin = stdRemove(a.mutedByAdmin, chatId)
			a.mu.Unlock()
		}
		break
	}
	return nil
}

func (a *Assistant) onGroupCall(m tg.Update, _ *tg.Client) error {
	updateGroupCall := m.(*tg.UpdateGroupCall)
	if groupCallRaw := updateGroupCall.Call; groupCallRaw != nil {
		var chatID int64
		var err error

		if updateGroupCall.Peer != nil {
			chatID, err = a.parseChatId(context.Background(), updateGroupCall.Peer)
			if err != nil {
				return err
			}
		} else {
			var callID int64
			switch call := groupCallRaw.(type) {
			case *tg.GroupCallObj:
				callID = call.ID
			case *tg.GroupCallDiscarded:
				callID = call.ID
			}

			if callID != 0 {
				a.mu.RLock()
				for id, inputCall := range a.inputGroupCalls {
					if obj, ok := inputCall.(*tg.InputGroupCallObj); ok && obj.ID == callID {
						chatID = id
						break
					}
				}
				a.mu.RUnlock()
			}
		}

		if chatID == 0 {
			return nil
		}

		switch groupCallRaw.(type) {
		case *tg.GroupCallObj:
			groupCall := groupCallRaw.(*tg.GroupCallObj)
			a.mu.Lock()
			a.inputGroupCalls[chatID] = &tg.InputGroupCallObj{
				ID:         groupCall.ID,
				AccessHash: groupCall.AccessHash,
			}
			a.mu.Unlock()
			return nil
		case *tg.GroupCallDiscarded:
			a.mu.Lock()
			delete(a.inputGroupCalls, chatID)
			a.mu.Unlock()
			_ = a.binding.Stop(chatID)
			return nil
		}
	}
	return nil
}

func (a *Assistant) onRequestBroadcastTimestamp(chatId int64) {
	a.mu.RLock()
	inputGroupCall := a.inputGroupCalls[chatId]
	a.mu.RUnlock()
	if inputGroupCall != nil {
		channels, err := a.App.PhoneGetGroupCallStreamChannels(inputGroupCall)
		if err == nil && len(channels.Channels) > 0 {
			_ = a.binding.SendBroadcastTimestamp(chatId, channels.Channels[0].LastTimestampMs)
		}
	}
}

func (a *Assistant) onRequestBroadcastPart(chatId int64, segmentPartRequest ntgcalls.SegmentPartRequest) {
	a.mu.RLock()
	inputGroupCall := a.inputGroupCalls[chatId]
	a.mu.RUnlock()
	if inputGroupCall != nil {
		file, err := a.App.UploadGetFile(
			&tg.UploadGetFileParams{
				Location: &tg.InputGroupCallStream{
					Call:         inputGroupCall,
					TimeMs:       segmentPartRequest.Timestamp,
					Scale:        0,
					VideoChannel: segmentPartRequest.ChannelID,
					VideoQuality: max(int32(segmentPartRequest.Quality), 0),
				},
				Offset: 0,
				Limit:  segmentPartRequest.Limit,
			},
		)

		status := ntgcalls.SegmentStatusNotReady
		var data []byte
		data = nil

		if err != nil {
			secondsWait := tg.GetFloodWait(err)
			if secondsWait == 0 {
				status = ntgcalls.SegmentStatusResyncNeeded
			}
		} else {
			data = file.(*tg.UploadFileObj).Bytes
			status = ntgcalls.SegmentStatusSuccess
		}

		_ = a.binding.SendBroadcastPart(
			chatId,
			segmentPartRequest.SegmentID,
			segmentPartRequest.PartID,
			status,
			segmentPartRequest.QualityUpdate,
			data,
		)
	}
}

func (a *Assistant) onConnectionChange(chatId int64, state ntgcalls.NetworkInfo) {
	a.mu.RLock()
	waitCh := a.waitConnect[chatId]
	a.mu.RUnlock()
	if waitCh == nil {
		return
	}

	var err error
	switch state.State {
	case ntgcalls.Connected:
		err = nil
	case ntgcalls.Closed, ntgcalls.Failed:
		err = fmt.Errorf("connection failed")
	case ntgcalls.Timeout:
		err = fmt.Errorf("connection timeout")
	default:
		return
	}

	select {
	case waitCh <- err:
	default:
	}
}

func (a *Assistant) onUpgrade(chatId int64, state ntgcalls.MediaState) {
	a.mu.RLock()
	inputGroupCall := a.inputGroupCalls[chatId]
	a.mu.RUnlock()
	if inputGroupCall == nil {
		return
	}
	if err := a.setCallStatus(context.Background(), inputGroupCall, state); err != nil {
		a.App.Log.Warnf("failed to update call status: %v", err)
	}
}

func (a *Assistant) convertGroupCallId(callId int64) (int64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for chatId, inputCallInterface := range a.inputGroupCalls {
		if inputCall, ok := inputCallInterface.(*tg.InputGroupCallObj); ok {
			if inputCall.ID == callId {
				return chatId, nil
			}
		}
	}
	return 0, fmt.Errorf("group call id %d not found", callId)
}

func getParticipantId(peer tg.Peer) int64 {
	var participantId int64
	switch chatObj := peer.(type) {
	case *tg.PeerUser:
		participantId = chatObj.UserID
	case *tg.PeerChannel:
		participantId = chatObj.ChannelID
	case *tg.PeerChat:
		participantId = chatObj.ChatID
	}
	return participantId
}

func stdRemove[T comparable](slice []T, val T) []T {
	return slices.DeleteFunc(slice, func(e T) bool {
		return e == val
	})
}
