package proto

type GetSecurityBarsOffset = GetSecurityBars
type GetSecurityBarsOffsetRequest = GetSecurityBarsRequest
type GetSecurityBarsOffsetReply = GetSecurityBarsReply

func NewGetSecurityBarsOffset(req *GetSecurityBarsOffsetRequest) *GetSecurityBarsOffset {
	obj := NewGetSecurityBars(req)
	obj.reqHeader.Method = KMSG_SECURITYBARS_OFFSET
	return obj
}
