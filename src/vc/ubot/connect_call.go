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
	defer func() {
		ctx.waitConnectMu.Lock()
		if ctx.waitConnect[chatId] != nil {
			delete(ctx.waitConnect, chatId)
		}
		ctx.waitConnectMu.Unlock()
	}()
	ctx.waitConnectMu.Lock()
	ctx.waitConnect[chatId] = make(chan error)
	ctx.waitConnectMu.Unlock()
	if chatId >= 0 {
		defer func() {
			ctx.p2pConfigsMu.Lock()
			if ctx.p2pConfigs[chatId] != nil {
				delete(ctx.p2pConfigs, chatId)
			}
			ctx.p2pConfigsMu.Unlock()
		}()
		ctx.p2pConfigsMu.RLock()
		p2pConfig := ctx.p2pConfigs[chatId]
		ctx.p2pConfigsMu.RUnlock()
		if p2pConfig == nil {
			p2pConfigs, err := ctx.getP2PConfigs(nil)
			if err != nil {
				return err
			}
			ctx.p2pConfigsMu.Lock()
			ctx.p2pConfigs[chatId] = p2pConfigs
			ctx.p2pConfigsMu.Unlock()
			p2pConfig = p2pConfigs
		}

		err := ctx.binding.CreateP2PCall(chatId)
		if err != nil {
			return err
		}

		err = ctx.binding.SetStreamSources(chatId, ntgcalls.CaptureStream, mediaDescription)
		if err != nil {
			return err
		}

		p2pConfig.GAorB, err = ctx.binding.InitExchange(chatId, ntgcalls.DhConfig{
			G:      p2pConfig.DhConfig.G,
			P:      p2pConfig.DhConfig.P,
			Random: p2pConfig.DhConfig.Random,
		}, p2pConfig.GAorB)
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
		if p2pConfig.IsOutgoing {
			_, err = ctx.App.PhoneRequestCall(
				&tg.PhoneRequestCallParams{
					Protocol: protocol,
					UserID:   userId,
					GAHash:   p2pConfig.GAorB,
					RandomID: int32(tg.GenRandInt()),
					Video:    mediaDescription.Camera != nil || mediaDescription.Screen != nil,
				},
			)
			if err != nil {
				return err
			}
		} else {
			ctx.inputCallsMu.RLock()
			inputCall := ctx.inputCalls[chatId]
			ctx.inputCallsMu.RUnlock()
			_, err = ctx.App.PhoneAcceptCall(
				inputCall,
				p2pConfig.GAorB,
				protocol,
			)
			if err != nil {
				return err
			}
		}
		select {
		case err = <-p2pConfig.WaitData:
			if err != nil {
				return err
			}
		case <-time.After(10 * time.Second):
			return fmt.Errorf("timed out waiting for an answer")
		}
		res, err := ctx.binding.ExchangeKeys(
			chatId,
			p2pConfig.GAorB,
			p2pConfig.KeyFingerprint,
		)
		if err != nil {
			return err
		}

		if p2pConfig.IsOutgoing {
			ctx.inputCallsMu.RLock()
			inputCall := ctx.inputCalls[chatId]
			ctx.inputCallsMu.RUnlock()
			confirmRes, err := ctx.App.PhoneConfirmCall(
				inputCall,
				res.GAOrB,
				res.KeyFingerprint,
				protocol,
			)
			if err != nil {
				return err
			}
			p2pConfig.PhoneCall = confirmRes.PhoneCall.(*tg.PhoneCallObj)
		}

		err = ctx.binding.ConnectP2P(
			chatId,
			parseRTCServers(p2pConfig.PhoneCall.Connections),
			p2pConfig.PhoneCall.Protocol.LibraryVersions,
			p2pConfig.PhoneCall.P2PAllowed,
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
			ctx.pendingConnectionsMu.Lock()
			ctx.pendingConnections[chatId] = &types.PendingConnection{
				MediaDescription: mediaDescription,
				Payload:          jsonParams,
			}
			ctx.pendingConnectionsMu.Unlock()
		}
	}
	ctx.waitConnectMu.RLock()
	waitConnect := ctx.waitConnect[chatId]
	ctx.waitConnectMu.RUnlock()
	return <-waitConnect
}
