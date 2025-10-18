package types

import (
	"tgmusic/pkg/vc/ntgcalls"
)

type PendingConnection struct {
	MediaDescription ntgcalls.MediaDescription
	Payload          string
	Presentation     bool
}
