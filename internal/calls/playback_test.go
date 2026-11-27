package calls

import (
	"errors"
	"reflect"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		err          error
		expectedKind errorKind
	}{
		{errors.New("group call is closed"), errFatal},
		{errors.New("GROUPCALL_FORBIDDEN"), errFatal},
		{errors.New("GROUPCALL_INVALID"), errFatal},
		{errors.New("GROUPCALL_ADD_PARTICIPANTS_FAILED"), errRetryOnce},
		{errors.New("CHANNELS_TOO_MUCH"), errRotate},
		{errors.New("FROZEN_METHOD_INVALID"), errRotate},
		{errors.New("FLOOD_WAIT_X"), errRotate},
		{errors.New("FLOOD_WAIT_10"), errRotate},
		{errors.New("FLOOD_PREMIUM_WAIT_10"), errRotate},
		{errors.New("USER_DEACTIVATED"), errRotate},
		{errors.New("some unknown error"), errUnknown},
	}

	for _, tt := range tests {
		kind := classifyError(tt.err)
		if kind != tt.expectedKind {
			t.Errorf("classifyError(%v) = %v, want %v", tt.err, kind, tt.expectedKind)
		}
	}
}

func TestPlayWithAssistantsRotatesAfterChannelLimit(t *testing.T) {
	first := &AssistantAccount{}
	second := &AssistantAccount{}
	c := &TelegramCalls{accounts: []*AssistantAccount{first, second}}
	joinErr := c.handleJoinError(nil, -100123, 123, 0, errors.New("[CHANNELS_TOO_MUCH] You have joined too many channels/supergroups"))
	if classifyError(joinErr) != errRotate {
		t.Fatalf("join error %v did not reach playback as a rotation error", joinErr)
	}

	var attempts, evicted []int
	saved := -1
	err := c.playWithAssistants(-100123, first, 0, playbackActions{
		attempt: func(call *AssistantAccount, index int) error {
			attempts = append(attempts, index)
			if index == 0 {
				if call != first {
					t.Fatal("first attempt used the wrong assistant")
				}
				return joinErr
			}
			if index != 1 || call != second {
				t.Fatalf("fallback used assistant %d, want assistant 1", index)
			}
			return nil
		},
		save: func(index int) { saved = index },
		evict: func(index int, err error) {
			if !errors.Is(err, joinErr) {
				t.Errorf("evicted assistant after %v, want channel limit error", err)
			}
			evicted = append(evicted, index)
		},
	})
	if err != nil {
		t.Fatalf("playWithAssistants() = %v, want fallback success", err)
	}
	if !reflect.DeepEqual(attempts, []int{0, 1}) {
		t.Errorf("attempts = %v, want [0 1]", attempts)
	}
	if !reflect.DeepEqual(evicted, []int{0}) {
		t.Errorf("evicted = %v, want [0]", evicted)
	}
	if saved != 1 {
		t.Errorf("saved assistant = %d, want 1", saved)
	}
}
