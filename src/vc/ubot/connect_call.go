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
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) connectCall(chatId int64, mediaDescription ntgcalls.MediaDescription, jsonParams string) error {
	chatMutex := ctx.getChatMutex(chatId)
	chatMutex.Lock()
	defer chatMutex.Unlock()

	ctx.stateMutex.Lock()
	if ctx.waitConnect[chatId] != nil {
		ctx.stateMutex.Unlock()
		return fmt.Errorf("connection already in progress for chat %d", chatId)
	}
	ctx.waitConnect[chatId] = make(chan error, 1) // Buffered to prevent deadlock
	ctx.stateMutex.Unlock()

	defer func() {
		ctx.stateMutex.Lock()
		if ctx.waitConnect[chatId] != nil {
			delete(ctx.waitConnect, chatId)
		}
		ctx.stateMutex.Unlock()
	}()
	if chatId >= 0 {
		defer func() {
			ctx.stateMutex.Lock()
			if ctx.p2pConfigs[chatId] != nil {
				delete(ctx.p2pConfigs, chatId)
			}
			ctx.stateMutex.Unlock()
		}()
		ctx.stateMutex.Lock()
		p2pConfig := ctx.p2pConfigs[chatId]
		ctx.stateMutex.Unlock()
		if p2pConfig == nil {
			p2pConfigs, err := ctx.getP2PConfigs(nil)
			if err != nil {
				return err
			}
			ctx.stateMutex.Lock()
			ctx.p2pConfigs[chatId] = p2pConfigs
			ctx.stateMutex.Unlock()
		}

		err := ctx.binding.CreateP2PCall(chatId)
		if err != nil {
			return err
		}

		err = ctx.binding.SetStreamSources(chatId, ntgcalls.CaptureStream, mediaDescription)
		if err != nil {
			return err
		}

		ctx.stateMutex.Lock()
		dhConfig := ctx.p2pConfigs[chatId].DhConfig
		gaOrB := ctx.p2pConfigs[chatId].GAorB
		ctx.stateMutex.Unlock()

		newGaOrB, err := ctx.binding.InitExchange(chatId, ntgcalls.DhConfig{
			G:      dhConfig.G,
			P:      dhConfig.P,
			Random: dhConfig.Random,
		}, gaOrB)
		if err == nil {
			ctx.stateMutex.Lock()
			ctx.p2pConfigs[chatId].GAorB = newGaOrB
			ctx.stateMutex.Unlock()
		}
		if err != nil {
			return err
		}

		protocolRaw := ntgcalls.GetProtocol()
		protocol := &tg.PhoneCallProtocol{
			UdpP2P:          protocolRaw.UdpP2P,
			UdpReflector:    protocolRaw.UdpReflector,
			MinLayer:        protocolRaw.MinLayer,
			MaxLayer:        protocolRaw.MaxLayer,
			LibraryVersions: protocolRaw.Versions,
		}

		userId, err := ctx.App.GetSendableUser(chatId)
		if err != nil {
			return err
		}
		ctx.stateMutex.Lock()
		isOutgoing := ctx.p2pConfigs[chatId].IsOutgoing
		ctx.stateMutex.Unlock()

		if isOutgoing {
			ctx.stateMutex.Lock()
			gaHash := ctx.p2pConfigs[chatId].GAorB
			ctx.stateMutex.Unlock()
			_, err = ctx.App.PhoneRequestCall(
				&tg.PhoneRequestCallParams{
					Protocol: protocol,
					UserID:   userId,
					GAHash:   gaHash,
					RandomID: int32(tg.GenRandInt()),
					Video:    mediaDescription.Camera != nil || mediaDescription.Screen != nil,
				},
			)
			if err != nil {
				return err
			}
		} else {
			ctx.stateMutex.Lock()
			inputCall := ctx.inputCalls[chatId]
			gaOrB := ctx.p2pConfigs[chatId].GAorB
			ctx.stateMutex.Unlock()
			_, err = ctx.App.PhoneAcceptCall(
				inputCall,
				gaOrB,
				protocol,
			)
			if err != nil {
				return err
			}
		}

		ctx.stateMutex.Lock()
		waitData := ctx.p2pConfigs[chatId].WaitData
		ctx.stateMutex.Unlock()

		select {
		case err = <-waitData:
			if err != nil {
				return err
			}
		case <-time.After(10 * time.Second):
			return fmt.Errorf("timed out waiting for an answer")
		}

		ctx.stateMutex.Lock()
		gaOrB = ctx.p2pConfigs[chatId].GAorB
		fingerprint := ctx.p2pConfigs[chatId].KeyFingerprint
		ctx.stateMutex.Unlock()
		res, err := ctx.binding.ExchangeKeys(
			chatId,
			gaOrB,
			fingerprint,
		)
		if err != nil {
			return err
		}

		ctx.stateMutex.Lock()
		isOutgoing = ctx.p2pConfigs[chatId].IsOutgoing
		ctx.stateMutex.Unlock()

		if isOutgoing {
			ctx.stateMutex.Lock()
			inputCall := ctx.inputCalls[chatId]
			ctx.stateMutex.Unlock()
			confirmRes, err := ctx.App.PhoneConfirmCall(
				inputCall,
				res.GAOrB,
				res.KeyFingerprint,
				protocol,
			)
			if err != nil {
				return err
			}
			ctx.stateMutex.Lock()
			ctx.p2pConfigs[chatId].PhoneCall = confirmRes.PhoneCall.(*tg.PhoneCallObj)
			ctx.stateMutex.Unlock()
		}

		ctx.stateMutex.Lock()
		phoneCall := ctx.p2pConfigs[chatId].PhoneCall
		ctx.stateMutex.Unlock()

		err = ctx.binding.ConnectP2P(
			chatId,
			parseRTCServers(phoneCall.Connections),
			phoneCall.Protocol.LibraryVersions,
			phoneCall.P2PAllowed,
		)
		if err != nil {
			return err
		}
	} else {
		var err error
		jsonParams, err = ctx.binding.CreateCall(chatId)
		if err != nil {
			_ = ctx.binding.Stop(chatId)
			return err
		}

		err = ctx.binding.SetStreamSources(chatId, ntgcalls.CaptureStream, mediaDescription)
		if err != nil {
			_ = ctx.binding.Stop(chatId)
			return err
		}

		inputGroupCall, err := ctx.getInputGroupCall(chatId)
		if err != nil {
			_ = ctx.binding.Stop(chatId)
			return err
		}

		resultParams := "{\"transport\": null}"
		callResRaw, err := ctx.App.PhoneJoinGroupCall(
			&tg.PhoneJoinGroupCallParams{
				Muted:        false,
				VideoStopped: mediaDescription.Camera == nil,
				Call:         inputGroupCall,
				Params: &tg.DataJson{
					Data: jsonParams,
				},
				JoinAs: &tg.InputPeerUser{
					UserID:     ctx.self.ID,
					AccessHash: ctx.self.AccessHash,
				},
			},
		)
		if err != nil {
			return err
		}
		callRes := callResRaw.(*tg.UpdatesObj)
		for _, update := range callRes.Updates {
			switch update.(type) {
			case *tg.UpdateGroupCallConnection:
				resultParams = update.(*tg.UpdateGroupCallConnection).Params.Data
			}
		}

		err = ctx.binding.Connect(
			chatId,
			resultParams,
			false,
		)
		if err != nil {
			return err
		}

		connectionMode, err := ctx.binding.GetConnectionMode(chatId)
		if err != nil {
			return err
		}

		if connectionMode == ntgcalls.StreamConnection && len(jsonParams) > 0 {
			ctx.stateMutex.Lock()
			ctx.pendingConnections[chatId] = &types.PendingConnection{
				MediaDescription: mediaDescription,
				Payload:          jsonParams,
			}
			ctx.stateMutex.Unlock()
		}
	}
	ctx.stateMutex.Lock()
	waitChan := ctx.waitConnect[chatId]
	ctx.stateMutex.Unlock()
	chatMutex.Unlock()
	err := <-waitChan
	chatMutex.Lock()
	return err
}
