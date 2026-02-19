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
			ctx.inputCalls[userId] = &tg.InputPhoneCall{
				ID:         ID,
				AccessHash: AccessHash,
			}
		}

		switch call := phoneCall.(type) {
		case *tg.PhoneCallAccepted:
			if ctx.p2pConfigs[userId] != nil {
				ctx.p2pConfigs[userId].GAorB = call.GB
				ctx.p2pConfigs[userId].WaitData <- nil
			}
		case *tg.PhoneCallObj:
			if ctx.p2pConfigs[userId] != nil {
				ctx.p2pConfigs[userId].GAorB = call.GAOrB
				ctx.p2pConfigs[userId].KeyFingerprint = call.KeyFingerprint
				ctx.p2pConfigs[userId].PhoneCall = call
				ctx.p2pConfigs[userId].WaitData <- nil
			}
		case *tg.PhoneCallDiscarded:
			var reasonMessage string
			switch call.Reason.(type) {
			case *tg.PhoneCallDiscardReasonBusy:
				reasonMessage = fmt.Sprintf("the user %d is busy", userId)
			case *tg.PhoneCallDiscardReasonHangup:
				reasonMessage = fmt.Sprintf("call declined by %d", userId)
			}
			if ctx.p2pConfigs[userId] != nil {
				ctx.p2pConfigs[userId].WaitData <- fmt.Errorf("%s", reasonMessage)
			}
			delete(ctx.inputCalls, userId)
			_ = ctx.binding.Stop(userId)
		case *tg.PhoneCallRequested:
			if ctx.p2pConfigs[userId] == nil {
				p2pConfigs, err := ctx.getP2PConfigs(call.GAHash)
				if err != nil {
					return err
				}
				ctx.p2pConfigs[userId] = p2pConfigs
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
			ctx.participantsMutex.Lock()
			if ctx.callParticipants[chatId] == nil {
				ctx.callParticipants[chatId] = &types.CallParticipantsCache{
					CallParticipants: make(map[int64]*tg.GroupCallParticipant),
				}
			}
			for _, participant := range participantsUpdate.Participants {
				participantId := getParticipantId(participant.Peer)
				if participant.Left {
					delete(ctx.callParticipants[chatId].CallParticipants, participantId)
					if ctx.callSources != nil && ctx.callSources[chatId] != nil {
						delete(ctx.callSources[chatId].CameraSources, participantId)
						delete(ctx.callSources[chatId].ScreenSources, participantId)
					}
					continue
				}

				ctx.callParticipants[chatId].CallParticipants[participantId] = participant
				if ctx.callSources != nil && ctx.callSources[chatId] != nil {
					wasCamera := ctx.callSources[chatId].CameraSources[participantId] != ""
					wasScreen := ctx.callSources[chatId].ScreenSources[participantId] != ""

					if wasCamera != (participant.Video != nil) {
						if participant.Video != nil {
							ctx.callSources[chatId].CameraSources[participantId] = participant.Video.Endpoint
							_, _ = ctx.binding.AddIncomingVideo(
								chatId,
								participant.Video.Endpoint,
								parseVideoSources(participant.Video.SourceGroups),
							)
						} else {
							_ = ctx.binding.RemoveIncomingVideo(
								chatId,
								ctx.callSources[chatId].CameraSources[participantId],
							)
							delete(ctx.callSources[chatId].CameraSources, participantId)
						}
					}

					if wasScreen != (participant.Presentation != nil) {
						if participant.Presentation != nil {
							ctx.callSources[chatId].ScreenSources[participantId] = participant.Presentation.Endpoint
							_, _ = ctx.binding.AddIncomingVideo(
								chatId,
								participant.Presentation.Endpoint,
								parseVideoSources(participant.Presentation.SourceGroups),
							)
						} else {
							_ = ctx.binding.RemoveIncomingVideo(
								chatId,
								ctx.callSources[chatId].ScreenSources[participantId],
							)
							delete(ctx.callSources[chatId].ScreenSources, participantId)
						}
					}
				}
			}
			ctx.callParticipants[chatId].LastMtprotoUpdate = time.Now()
			ctx.participantsMutex.Unlock()

			for _, participant := range participantsUpdate.Participants {
				participantId := getParticipantId(participant.Peer)
				if participantId == ctx.self.ID {
					connectionMode, err := ctx.binding.GetConnectionMode(chatId)
					if err == nil && connectionMode == ntgcalls.StreamConnection && participant.CanSelfUnmute {
						if ctx.pendingConnections[chatId] != nil {
							_ = ctx.connectCall(
								chatId,
								ctx.pendingConnections[chatId].MediaDescription,
								ctx.pendingConnections[chatId].Payload,
							)
						}
					} else if !participant.CanSelfUnmute {
						if !slices.Contains(ctx.mutedByAdmin, chatId) {
							ctx.mutedByAdmin = append(ctx.mutedByAdmin, chatId)
						}
					} else if slices.Contains(ctx.mutedByAdmin, chatId) {
						state, err := ctx.binding.GetState(chatId)
						if err != nil {
							panic(err)
						}
						err = ctx.setCallStatus(participantsUpdate.Call, state)
						if err != nil {
							panic(err)
						}
						ctx.mutedByAdmin = stdRemove(ctx.mutedByAdmin, chatId)
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
		_, _ = ctx.App.PhoneSendSignalingData(ctx.inputCalls[chatId], signal)
	})

	ctx.binding.OnConnectionChange(func(chatId int64, state ntgcalls.NetworkInfo) {
		if ctx.waitConnect[chatId] != nil {
			switch state.State {
			case ntgcalls.Connected:
				ctx.waitConnect[chatId] <- nil
			case ntgcalls.Closed, ntgcalls.Failed:
				ctx.waitConnect[chatId] <- fmt.Errorf("connection failed")
			case ntgcalls.Timeout:
				ctx.waitConnect[chatId] <- fmt.Errorf("connection timeout")
			default:
			}
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
}
