package proto

type GetSecurityBarsOffset = GetSecurityBars
type GetSecurityBarsOffsetRequest = GetSecurityBarsRequest
type GetSecurityBarsOffsetReply = GetSecurityBarsReply

func NewGetSecurityBarsOffset() *GetSecurityBarsOffset {
	obj := NewGetSecurityBars()
	obj.reqHeader.Method = KMSG_SECURITYBARS_OFFSET
	return obj
}
