package calls

import (
	"errors"
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
