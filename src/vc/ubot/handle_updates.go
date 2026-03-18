/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

import (
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"ashokshau/tgmusic/src/vc/ubot/types"
	"fmt"
	"slices"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) handleUpdates() {
	ctx.App.AddRawHandler(&tg.UpdatePhoneCallSignalingData{}, func(m tg.Update, c *tg.Client) error {
		signalingData := m.(*tg.UpdatePhoneCallSignalingData)
		userId, err := ctx.convertCallId(signalingData.PhoneCallID)
		if err == nil {
			_ = ctx.binding.SendSignalingData(userId, signalingData.Data)
		}
		return nil
	})

	ctx.App.AddRawHandler(&tg.UpdatePhoneCall{}, func(m tg.Update, _ *tg.Client) error {
		phoneCall := m.(*tg.UpdatePhoneCall).PhoneCall

		var ID int64
		var AccessHash int64
		var userId int64

		switch call := phoneCall.(type) {
		case *tg.PhoneCallAccepted:
			userId = call.ParticipantID
			ID = call.ID
			AccessHash = call.AccessHash
		case *tg.PhoneCallWaiting:
			userId = call.ParticipantID
			ID = call.ID
			AccessHash = call.AccessHash
		case *tg.PhoneCallRequested:
			userId = call.AdminID
			ID = call.ID
			AccessHash = call.AccessHash
		case *tg.PhoneCallObj:
			userId = call.AdminID
		case *tg.PhoneCallDiscarded:
			userId, _ = ctx.convertCallId(call.ID)
		}

		switch phoneCall.(type) {
		case *tg.PhoneCallAccepted, *tg.PhoneCallRequested, *tg.PhoneCallWaiting:
			ctx.stateMutex.Lock()
			ctx.inputCalls[userId] = &tg.InputPhoneCall{
				ID:         ID,
				AccessHash: AccessHash,
			}
			ctx.stateMutex.Unlock()
		}

		switch call := phoneCall.(type) {
		case *tg.PhoneCallAccepted:
			ctx.stateMutex.Lock()
			p2pConfig := ctx.p2pConfigs[userId]
			if p2pConfig != nil {
				p2pConfig.GAorB = call.GB
			}
			ctx.stateMutex.Unlock()
			if p2pConfig != nil {
				p2pConfig.WaitData <- nil
			}
		case *tg.PhoneCallObj:
			ctx.stateMutex.Lock()
			p2pConfig := ctx.p2pConfigs[userId]
			if p2pConfig != nil {
				p2pConfig.GAorB = call.GAOrB
				p2pConfig.KeyFingerprint = call.KeyFingerprint
				p2pConfig.PhoneCall = call
			}
			ctx.stateMutex.Unlock()
			if p2pConfig != nil {
				p2pConfig.WaitData <- nil
			}
		case *tg.PhoneCallDiscarded:
			var reasonMessage string
			switch call.Reason.(type) {
			case *tg.PhoneCallDiscardReasonBusy:
				reasonMessage = fmt.Sprintf("the user %d is busy", userId)
			case *tg.PhoneCallDiscardReasonHangup:
				reasonMessage = fmt.Sprintf("call declined by %d", userId)
			}
			ctx.stateMutex.Lock()
			p2pConfig := ctx.p2pConfigs[userId]
			delete(ctx.inputCalls, userId)
			ctx.stateMutex.Unlock()
			if p2pConfig != nil {
				p2pConfig.WaitData <- fmt.Errorf("%s", reasonMessage)
			}
			_ = ctx.binding.Stop(userId)
		case *tg.PhoneCallRequested:
			ctx.stateMutex.Lock()
			p2pConfig := ctx.p2pConfigs[userId]
			ctx.stateMutex.Unlock()
			if p2pConfig == nil {
				p2pConfigs, err := ctx.getP2PConfigs(call.GAHash)
				if err != nil {
					return err
				}
				ctx.stateMutex.Lock()
				ctx.p2pConfigs[userId] = p2pConfigs
				ctx.stateMutex.Unlock()
				for _, callback := range ctx.incomingCallCallbacks {
					go callback(ctx, userId)
				}
			}
		}
		return nil
	})

	ctx.App.AddRawHandler(&tg.UpdateGroupCallParticipants{}, func(m tg.Update, c *tg.Client) error {
		participantsUpdate := m.(*tg.UpdateGroupCallParticipants)
		chatId, err := ctx.convertGroupCallId(participantsUpdate.Call.(*tg.InputGroupCallObj).ID)
		if err == nil {
			chatMutex := ctx.getChatMutex(chatId)
			chatMutex.Lock()
			ctx.stateMutex.Lock()
			if ctx.callParticipants[chatId] == nil {
				ctx.callParticipants[chatId] = &types.CallParticipantsCache{
					CallParticipants: make(map[int64]*tg.GroupCallParticipant),
				}
			}
			ctx.stateMutex.Unlock()
			for _, participant := range participantsUpdate.Participants {
				participantId := getParticipantId(participant.Peer)
				if participant.Left {
					ctx.stateMutex.Lock()
					delete(ctx.callParticipants[chatId].CallParticipants, participantId)
					if ctx.callSources != nil && ctx.callSources[chatId] != nil {
						delete(ctx.callSources[chatId].CameraSources, participantId)
						delete(ctx.callSources[chatId].ScreenSources, participantId)
					}
					ctx.stateMutex.Unlock()
					continue
				}

				ctx.stateMutex.Lock()
				ctx.callParticipants[chatId].CallParticipants[participantId] = participant
				if ctx.callSources != nil && ctx.callSources[chatId] != nil {
					wasCamera := ctx.callSources[chatId].CameraSources[participantId] != ""
					wasScreen := ctx.callSources[chatId].ScreenSources[participantId] != ""
					ctx.stateMutex.Unlock()

					if wasCamera != (participant.Video != nil) {
						if participant.Video != nil {
							ctx.stateMutex.Lock()
							ctx.callSources[chatId].CameraSources[participantId] = participant.Video.Endpoint
							ctx.stateMutex.Unlock()
							_, _ = ctx.binding.AddIncomingVideo(
								chatId,
								participant.Video.Endpoint,
								parseVideoSources(participant.Video.SourceGroups),
							)
						} else {
							ctx.stateMutex.Lock()
							endpoint := ctx.callSources[chatId].CameraSources[participantId]
							ctx.stateMutex.Unlock()
							_ = ctx.binding.RemoveIncomingVideo(
								chatId,
								endpoint,
							)
							ctx.stateMutex.Lock()
							delete(ctx.callSources[chatId].CameraSources, participantId)
							ctx.stateMutex.Unlock()
						}
					}

					if wasScreen != (participant.Presentation != nil) {
						if participant.Presentation != nil {
							ctx.stateMutex.Lock()
							ctx.callSources[chatId].ScreenSources[participantId] = participant.Presentation.Endpoint
							ctx.stateMutex.Unlock()
							_, _ = ctx.binding.AddIncomingVideo(
								chatId,
								participant.Presentation.Endpoint,
								parseVideoSources(participant.Presentation.SourceGroups),
							)
						} else {
							ctx.stateMutex.Lock()
							endpoint := ctx.callSources[chatId].ScreenSources[participantId]
							ctx.stateMutex.Unlock()
							_ = ctx.binding.RemoveIncomingVideo(
								chatId,
								endpoint,
							)
							ctx.stateMutex.Lock()
							delete(ctx.callSources[chatId].ScreenSources, participantId)
							ctx.stateMutex.Unlock()
						}
					}
				} else {
					ctx.stateMutex.Unlock()
				}
			}
			ctx.stateMutex.Lock()
			ctx.callParticipants[chatId].LastMtprotoUpdate = time.Now()
			ctx.stateMutex.Unlock()
			chatMutex.Unlock()

			for _, participant := range participantsUpdate.Participants {
				participantId := getParticipantId(participant.Peer)
				if participantId == ctx.self.ID {
					connectionMode, err := ctx.binding.GetConnectionMode(chatId)
					if err == nil && connectionMode == ntgcalls.StreamConnection && participant.CanSelfUnmute {
						ctx.stateMutex.Lock()
						pendingConn := ctx.pendingConnections[chatId]
						ctx.stateMutex.Unlock()
						if pendingConn != nil {
							_ = ctx.connectCall(
								chatId,
								pendingConn.MediaDescription,
								pendingConn.Payload,
							)
						}
					} else if !participant.CanSelfUnmute {
						ctx.stateMutex.Lock()
						if !slices.Contains(ctx.mutedByAdmin, chatId) {
							ctx.mutedByAdmin = append(ctx.mutedByAdmin, chatId)
						}
						ctx.stateMutex.Unlock()
					} else {
						ctx.stateMutex.Lock()
						isMutedByAdmin := slices.Contains(ctx.mutedByAdmin, chatId)
						ctx.stateMutex.Unlock()
						if isMutedByAdmin {
							state, err := ctx.binding.GetState(chatId)
							if err != nil {
								panic(err)
							}
							err = ctx.setCallStatus(participantsUpdate.Call, state)
							if err != nil {
								panic(err)
							}
							ctx.stateMutex.Lock()
							ctx.mutedByAdmin = stdRemove(ctx.mutedByAdmin, chatId)
							ctx.stateMutex.Unlock()
						}
					}
				}
			}
		}
		return nil
	})

	ctx.App.AddRawHandler(&tg.UpdateGroupCall{}, func(m tg.Update, c *tg.Client) error {
		updateGroupCall := m.(*tg.UpdateGroupCall)
		if groupCallRaw := updateGroupCall.Call; groupCallRaw != nil {
			var chatID int64
			var err error

			if updateGroupCall.Peer != nil {
				chatID, err = ctx.parseChatId(updateGroupCall.Peer)
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
					ctx.inputGroupCallsMutex.RLock()
					for id, inputCall := range ctx.inputGroupCalls {
						if obj, ok := inputCall.(*tg.InputGroupCallObj); ok && obj.ID == callID {
							chatID = id
							ctx.App.Log.Debugf("Received UpdateGroupCall with nil Peer and resolved:%v", chatID)
							break
						}
					}
					ctx.inputGroupCallsMutex.RUnlock()
				}
			}

			if chatID == 0 {
				/*
					raw, _ := json.MarshalIndent(m, "", "  ")
					ctx.App.Log.Debugf("Received UpdateGroupCall with nil Peer and unknown call ID:%s", string(raw))
				*/
				return nil
			}

			switch groupCallRaw.(type) {
			case *tg.GroupCallObj:
				groupCall := groupCallRaw.(*tg.GroupCallObj)
				ctx.inputGroupCallsMutex.Lock()
				ctx.inputGroupCalls[chatID] = &tg.InputGroupCallObj{
					ID:         groupCall.ID,
					AccessHash: groupCall.AccessHash,
				}
				ctx.inputGroupCallsMutex.Unlock()
				return nil
			case *tg.GroupCallDiscarded:
				ctx.inputGroupCallsMutex.Lock()
				delete(ctx.inputGroupCalls, chatID)
				ctx.inputGroupCallsMutex.Unlock()
				_ = ctx.binding.Stop(chatID)
				return nil
			default:
				ctx.App.Log.Warnf("Received UpdateGroupCall with unknown type:%v", groupCallRaw)
			}
		}
		return nil
	})

	ctx.binding.OnRequestBroadcastTimestamp(func(chatId int64) {
		ctx.inputGroupCallsMutex.RLock()
		inputGroupCall := ctx.inputGroupCalls[chatId]
		ctx.inputGroupCallsMutex.RUnlock()
		if inputGroupCall != nil {
			channels, err := ctx.App.PhoneGetGroupCallStreamChannels(inputGroupCall)
			if err == nil {
				_ = ctx.binding.SendBroadcastTimestamp(chatId, channels.Channels[0].LastTimestampMs)
			}
		}
	})

	ctx.binding.OnRequestBroadcastPart(func(chatId int64, segmentPartRequest ntgcalls.SegmentPartRequest) {
		ctx.inputGroupCallsMutex.RLock()
		inputGroupCall := ctx.inputGroupCalls[chatId]
		ctx.inputGroupCallsMutex.RUnlock()
		if inputGroupCall != nil {
			file, err := ctx.App.UploadGetFile(
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

			_ = ctx.binding.SendBroadcastPart(
				chatId,
				segmentPartRequest.SegmentID,
				segmentPartRequest.PartID,
				status,
				segmentPartRequest.QualityUpdate,
				data,
			)
		}
	})

	ctx.binding.OnSignal(func(chatId int64, signal []byte) {
		ctx.stateMutex.Lock()
		inputCall := ctx.inputCalls[chatId]
		ctx.stateMutex.Unlock()
		_, _ = ctx.App.PhoneSendSignalingData(inputCall, signal)
	})

	ctx.binding.OnConnectionChange(func(chatId int64, state ntgcalls.NetworkInfo) {
		ctx.stateMutex.Lock()
		waitChan := ctx.waitConnect[chatId]
		ctx.stateMutex.Unlock()
		if waitChan != nil {
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
			waitChan <- err
		}
	})

	ctx.binding.OnUpgrade(func(chatId int64, state ntgcalls.MediaState) {
		ctx.inputGroupCallsMutex.RLock()
		inputGroupCall := ctx.inputGroupCalls[chatId]
		ctx.inputGroupCallsMutex.RUnlock()
		err := ctx.setCallStatus(inputGroupCall, state)
		if err != nil {
			fmt.Println(err)
		}
	})

	ctx.binding.OnStreamEnd(func(chatId int64, streamType ntgcalls.StreamType, streamDevice ntgcalls.StreamDevice) {
		for _, callback := range ctx.streamEndCallbacks {
			go callback(chatId, streamType, streamDevice)
		}
	})

	ctx.binding.OnFrame(func(chatId int64, mode ntgcalls.StreamMode, device ntgcalls.StreamDevice, frames []ntgcalls.Frame) {
		for _, callback := range ctx.frameCallbacks {
			go callback(chatId, mode, device, frames)
		}
	})
}
