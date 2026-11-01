package proto

// GetSecurityBarsOffset 是偏移量形式 K 线查询流程的类型别名，复用 GetSecurityBars 的全部实现。
//
// 偏移 K 线与普通 K 线共用同一套请求参数、报文编码与响应解码逻辑，仅在下行读取时
// 按偏移定位窗口；使用方通过 NewGetSecurityBarsOffset 构造，不要直接复制零值。
type GetSecurityBarsOffset = GetSecurityBars

// GetSecurityBarsOffsetRequest 是偏移量形式 K 线查询请求参数的类型别名。
type GetSecurityBarsOffsetRequest = GetSecurityBarsRequest

// GetSecurityBarsOffsetReply 是偏移量形式 K 线查询响应数据的类型别名。
type GetSecurityBarsOffsetReply = GetSecurityBarsReply

// NewGetSecurityBarsOffset 构造偏移量形式 K 线查询对象，并把协议命令字切换为偏移 K 线指令。
//
// 参数 `req` 指定市场、证券代码、K 线周期、条数与起始偏移等普通 K 线请求参数，
// 与 GetSecurityBarsRequest 同构；`req` 为 `nil` 时使用空请求参数。构造时先复用
// 普通 K 线的构造流程完成包头初始化与参数绑定，再覆盖请求头中的命令字；报文编码
// 与响应解码沿用 GetSecurityBars 的既有方法。
func NewGetSecurityBarsOffset(req *GetSecurityBarsOffsetRequest) *GetSecurityBarsOffset {
	obj := NewGetSecurityBars(req)
	// 仅替换命令字为偏移 K 线指令，其余报文格式与普通 K 线保持一致。
	obj.reqHeader.Method = KMSG_SECURITYBARS_OFFSET
	return obj
}
