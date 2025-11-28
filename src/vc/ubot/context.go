package ubot

import (
	"ashokshau/tgmusic/src/vc/ntgcalls"
	"ashokshau/tgmusic/src/vc/ubot/types"
	"sync"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type Context struct {
	binding               *ntgcalls.Client
	App                   *tg.Client
	mutedByAdmin          []int64
	presentations         []int64
	presentationsMutex    sync.RWMutex
	p2pConfigs            map[int64]*types.P2PConfig
	p2pMutex              sync.RWMutex
	inputCalls            map[int64]*tg.InputPhoneCall
	inputCallsMutex       sync.RWMutex
	inputGroupCalls       map[int64]tg.InputGroupCall
	groupCallsMutex       sync.RWMutex
	participantsMutex     sync.Mutex
	callParticipants      map[int64]*types.CallParticipantsCache
	pendingConnections    map[int64]*types.PendingConnection
	pendingConnMutex      sync.RWMutex
	callSources           map[int64]*types.CallSources
	waitConnect           map[int64]chan error
	waitConnMutex         sync.RWMutex
	self                  *tg.UserObj
	incomingCallCallbacks []func(client *Context, chatId int64)
	streamEndCallbacks    []ntgcalls.StreamEndCallback
	frameCallbacks        []ntgcalls.FrameCallback
}

func NewInstance(app *tg.Client) (*Context, error) {
	client := &Context{
		binding:            ntgcalls.NTgCalls(),
		App:                app,
		p2pConfigs:         make(map[int64]*types.P2PConfig),
		inputCalls:         make(map[int64]*tg.InputPhoneCall),
		inputGroupCalls:    make(map[int64]tg.InputGroupCall),
		pendingConnections: make(map[int64]*types.PendingConnection),
		callParticipants:   make(map[int64]*types.CallParticipantsCache),
		callSources:        make(map[int64]*types.CallSources),
		waitConnect:        make(map[int64]chan error),
	}

	if app.IsConnected() {
		self, err := app.GetMe()
		if err != nil {
			client.Close()
			return nil, err
		}
		client.self = self
	}

	client.handleUpdates()
	return client, nil
}

func (ctx *Context) OnIncomingCall(callback func(client *Context, chatId int64)) {
	ctx.incomingCallCallbacks = append(ctx.incomingCallCallbacks, callback)
}

func (ctx *Context) OnStreamEnd(callback ntgcalls.StreamEndCallback) {
	ctx.streamEndCallbacks = append(ctx.streamEndCallbacks, callback)
}

func (ctx *Context) OnFrame(callback ntgcalls.FrameCallback) {
	ctx.frameCallbacks = append(ctx.frameCallbacks, callback)
}

func (ctx *Context) Close() {
	ctx.binding.Free()
}
