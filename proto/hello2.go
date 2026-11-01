package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Hello2 表示主站第二次握手协议，负责构造登录请求、解析提示文案、下载链接与可选附加消息。
//
// 该协议请求携带固定登录负载；协议对象内部保存已解析结果，非并发安全，
// 不应在多个协程间共享同一实例。
type Hello2 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *Hello2Request
	reply      *Hello2Reply

	contentHex string
}

// Hello2Request 是第二次握手的请求参数；该协议无需携带业务参数，结构体仅用于占位。
type Hello2Request struct {
}

// Hello2Reply 保存第二次握手的响应结果，包含状态位、提示文案、下载链接与可选附加消息。
// 标注为待确认的字段按原始数值与十六进制片段原样保留，不做语义解读。
type Hello2Reply struct {
	Info        string // 下载链接起点之后的整段原始文本，由链接与可选附加消息拼接而成，已按 GBK 解码。
	Had         uint8  // 是否存在附加消息的状态位；取 0x01 时尾部携带附加消息。
	Unknown2    uint16 // 语义待确认，第一至二字节的数值，保留原始值。
	Tips        string // 提示文案，第三至五十二字节定长段，已按 GBK 解码。
	Link        string // 下载链接，第五十八至一百七十七字节定长段，已按 GBK 解码。
	Msg         string // 状态位置位时尾部追加的附加消息，已按 GBK 解码；无附加消息时保持空串。
	Unknown5Hex string // 语义待确认，第五十三至五十七字节的原始十六进制片段。
}

// NewHello2 创建第二次握手协议对象，并初始化默认请求头与固定登录负载；无入参，
// 返回已就绪的协议对象，对象内部状态非并发安全。
func NewHello2() *Hello2 {
	obj := new(Hello2)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(Hello2Request)
	obj.reply = new(Hello2Reply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_CMD2
	// 固定登录负载为协议规定的客户端标识串与填充字节，由服务端协议固定下发。
	obj.contentHex = "d5d0c9ccd6a4a8af0000008fc22540130000d500c9ccbdf0d7ea00000002"
	return obj
}

// BuildRequest 把请求头与固定登录负载编码为协议字节流；编码失败时返回非空错误。
//
// 报文体长度为两字节头部长度加固定负载字节数，两个长度字段互为冗余校验。
func (obj *Hello2) BuildRequest() ([]byte, error) {
	b, err := hex.DecodeString(obj.contentHex)

	// 长度字段记录两字节头部加上固定负载的总长度。
	obj.reqHeader.PkgLen1 = 2 + uint16(len(b))
	obj.reqHeader.PkgLen2 = 2 + uint16(len(b))

	buf := new(bytes.Buffer)
	// 请求头写入，随后追加固定登录负载。
	err = binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	buf.Write(b)
	return buf.Bytes(), err
}

// ParseResponse 解析第二次握手响应负载并填充内部响应结果；响应不足定长下限时返回错误。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节流；解析成功返回 `nil`，
// 主体长度不足一百七十八字节时返回非空错误。
func (obj *Hello2) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 响应主体不足定长下限一百七十八字节即视为非法握手响应，先拦截再切片。
	if len(data) < 178 {
		return fmt.Errorf("invalid hello2 response length: %d", len(data))
	}

	// 固定偏移布局：第零字节为状态位，第一至二字节为待确认数值，第三至五十二字节为提示文案，
	// 第五十三至五十七字节为待确认十六进制片段，第五十八至一百七十七字节为下载链接，文本段统一按 GBK 解码。
	obj.reply.Had = data[0]
	obj.reply.Unknown2 = binary.LittleEndian.Uint16(data[1:3])
	obj.reply.Tips = Utf8ToGbk(data[3:53])
	obj.reply.Unknown5Hex = hex.EncodeToString(data[53:58])
	obj.reply.Link = Utf8ToGbk(data[58:178])
	// 状态位置位且报文超出主体长度时，第一百七十八字节之后的剩余字节为附加消息。
	if obj.reply.Had == 0x01 && len(data) > 178 {
		obj.reply.Msg = Utf8ToGbk(data[178:])
	}
	obj.reply.Info = Utf8ToGbk(data[58:])
	return nil
}

// Response 返回解析完成后的第二次握手响应结果。
func (obj *Hello2) Response() *Hello2Reply {
	return obj.reply
}
