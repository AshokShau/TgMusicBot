package types

import "ashokshau/tgmusic/src/vc/ntgcalls"

type PendingConnection struct {
	MediaDescription ntgcalls.MediaDescription
	Payload          string
	Presentation     bool
}
