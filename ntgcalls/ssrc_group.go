package ntgcalls

// SsrcGroup represents an SSRC group with semantics and associated SSRC values.
type SsrcGroup struct {
	Semantics string
	Ssrcs     []uint32
}
