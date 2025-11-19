package ubot

import (
	"fmt"
	"time"

	"ashokshau/tgmusic/src/vc/ntgcalls"
	"ashokshau/tgmusic/src/vc/ubot/types"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func (ctx *Context) connectCall(chatId int64, mediaDescription ntgcalls.MediaDescription, jsonParams string) error {
	defer func() {
		ctx.waitConnMutex.Lock()
		if ctx.waitConnect[chatId] != nil {
			delete(ctx.waitConnect, chatId)
		}
		ctx.waitConnMutex.Unlock()
	}()
	ctx.waitConnMutex.Lock()
	ctx.waitConnect[chatId] = make(chan error, 1)
	ctx.waitConnMutex.Unlock()
	if chatId >= 0 {
		defer func() {
			ctx.p2pMutex.Lock()
			if ctx.p2pConfigs[chatId] != nil {
				delete(ctx.p2pConfigs, chatId)
			}
			ctx.p2pMutex.Unlock()
		}()
		ctx.p2pMutex.RLock()
		cfgNil := ctx.p2pConfigs[chatId] == nil
		ctx.p2pMutex.RUnlock()
		if cfgNil {
			p2pConfigs, err := ctx.getP2PConfigs(nil)
			if err != nil {
				return err
			}
			ctx.p2pMutex.Lock()
			ctx.p2pConfigs[chatId] = p2pConfigs
			ctx.p2pMutex.Unlock()
		}

		err := ctx.binding.CreateP2PCall(chatId)
		if err != nil {
			return err
		}

		err = ctx.binding.SetStreamSources(chatId, ntgcalls.CaptureStream, mediaDescription)
		if err != nil {
			return err
		}

		ctx.p2pMutex.RLock()
		dh := ctx.p2pConfigs[chatId].DhConfig
		gaorb := ctx.p2pConfigs[chatId].GAorB
		ctx.p2pMutex.RUnlock()
		newGAorB, err := ctx.binding.InitExchange(chatId, ntgcalls.DhConfig{
			G:      dh.G,
			P:      dh.P,
			Random: dh.Random,
		}, gaorb)
		if err != nil {
			return err
		}
		ctx.p2pMutex.Lock()
		ctx.p2pConfigs[chatId].GAorB = newGAorB
		ctx.p2pMutex.Unlock()

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
		ctx.p2pMutex.RLock()
		isOutgoing := ctx.p2pConfigs[chatId].IsOutgoing
		ctx.p2pMutex.RUnlock()
		if isOutgoing {
			_, err = ctx.App.PhoneRequestCall(
				&tg.PhoneRequestCallParams{
					Protocol: protocol,
					UserID:   userId,
					GAHash:   ctx.p2pConfigs[chatId].GAorB,
					RandomID: int32(tg.GenRandInt()),
					Video:    mediaDescription.Camera != nil || mediaDescription.Screen != nil,
				},
			)
			if err != nil {
				return err
			}
		} else {
			ctx.inputCallsMutex.RLock()
			inCall := ctx.inputCalls[chatId]
			ctx.inputCallsMutex.RUnlock()
			ctx.p2pMutex.RLock()
			gaorb := ctx.p2pConfigs[chatId].GAorB
			ctx.p2pMutex.RUnlock()
			_, err = ctx.App.PhoneAcceptCall(
				inCall,
				gaorb,
				protocol,
			)
			if err != nil {
				return err
			}
		}
		ctx.p2pMutex.RLock()
		waitData := ctx.p2pConfigs[chatId].WaitData
		ctx.p2pMutex.RUnlock()
		select {
		case err = <-waitData:
			if err != nil {
				return err
			}
		case <-time.After(10 * time.Second):
			return fmt.Errorf("timed out waiting for an answer")
		}
		ctx.p2pMutex.RLock()
		gaorb = ctx.p2pConfigs[chatId].GAorB
		keyFp := ctx.p2pConfigs[chatId].KeyFingerprint
		ctx.p2pMutex.RUnlock()
		res, err := ctx.binding.ExchangeKeys(
			chatId,
			gaorb,
			keyFp,
		)
		if err != nil {
			return err
		}

		ctx.p2pMutex.RLock()
		isOutgoing = ctx.p2pConfigs[chatId].IsOutgoing
		ctx.p2pMutex.RUnlock()
		if isOutgoing {
			ctx.inputCallsMutex.RLock()
			inCall := ctx.inputCalls[chatId]
			ctx.inputCallsMutex.RUnlock()
			confirmRes, err := ctx.App.PhoneConfirmCall(
				inCall,
				res.GAOrB,
				res.KeyFingerprint,
				protocol,
			)
			if err != nil {
				return err
			}
			ctx.p2pMutex.Lock()
			ctx.p2pConfigs[chatId].PhoneCall = confirmRes.PhoneCall.(*tg.PhoneCallObj)
			ctx.p2pMutex.Unlock()
		}

		ctx.p2pMutex.RLock()
		pc := ctx.p2pConfigs[chatId].PhoneCall
		ctx.p2pMutex.RUnlock()
		err = ctx.binding.ConnectP2P(
			chatId,
			parseRTCServers(pc.Connections),
			pc.Protocol.LibraryVersions,
			pc.P2PAllowed,
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
			ctx.pendingConnMutex.Lock()
			ctx.pendingConnections[chatId] = &types.PendingConnection{
				MediaDescription: mediaDescription,
				Payload:          jsonParams,
			}
			ctx.pendingConnMutex.Unlock()
		}
	}
	ctx.waitConnMutex.RLock()
	ch := ctx.waitConnect[chatId]
	ctx.waitConnMutex.RUnlock()
	select {
	case err := <-ch:
		return err
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timed out waiting for connection")
	}
}
