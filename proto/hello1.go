package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
)

// Hello1 表示主站首次握手协议，负责构造握手请求、解析服务端时间与站点信息并对外提供结果。
//
// 该协议请求固定携带握手负载；协议对象内部保存已解析结果，非并发安全，
// 不应在多个协程间共享同一实例。
type Hello1 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *Hello1Request
	reply      *Hello1Reply

	contentHex string
}

// Hello1Request 是首次握手的请求参数；该协议无需携带业务参数，结构体仅用于占位。
type Hello1Request struct {
}

// Hello1Reply 保存首次握手的响应结果，包含服务端时间、服务端名称、网址、类别文案与整段原始文本。
type Hello1Reply struct {
	Info       string // 第六十七字节之后的整段原始文本，由名称、网址、类别文本段拼接而成，已按 GBK 解码。
	DateTime   string // 服务端时间，格式为年-月-日 时:分:秒，按本地时区组合。
	ServerName string // 服务端名称，第六十七至八十八字节定长段，已按 GBK 解码。
	Website    string // 服务端网址，第八十九至一百五十二字节定长段，已按 GBK 解码。
	Category   string // 类别文案，第一百五十九至一百八十八字节定长段，已按 GBK 解码。
}

// NewHello1 创建主站首次握手协议对象，并初始化默认请求头与固定握手负载；无入参，
// 返回已就绪的协议对象，对象内部状态非并发安全。
func NewHello1() *Hello1 {
	obj := new(Hello1)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(Hello1Request)
	obj.reply = new(Hello1Reply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_CMD1
	// 固定握手负载为一，不携带其他业务参数。
	obj.contentHex = "01"
	return obj
}

// BuildRequest 把请求头与固定握手负载编码为协议字节流；编码失败时返回非空错误。
//
// 报文体长度为两字节头部长度加固定负载字节数，两个长度字段互为冗余校验。
func (obj *Hello1) BuildRequest() ([]byte, error) {
	b, err := hex.DecodeString(obj.contentHex)

	// 长度字段记录两字节头部加上固定负载的总长度。
	obj.reqHeader.PkgLen1 = 2 + uint16(len(b))
	obj.reqHeader.PkgLen2 = 2 + uint16(len(b))

	buf := new(bytes.Buffer)
	// 请求头写入，随后追加固定握手负载。
	err = binary.Write(buf, binary.LittleEndian, obj.reqHeader)

	buf.Write(b)
	return buf.Bytes(), err
}

// ParseResponse 解析首次握手响应负载并填充内部响应结果；响应不足定长下限时返回错误。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节流；解析成功返回 `nil`，
// 负载长度不足一百八十九字节时返回非空错误。
func (obj *Hello1) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 响应不足定长下限一百八十九字节即视为非法握手响应，先拦截再切片。
	if len(data) < 189 {
		return fmt.Errorf("invalid hello1 response length: %d", len(data))
	}

	// 服务端时间固定偏移：第一至三字节为十六位年份，第三字节为日，第四字节为月，
	// 第五字节为分，第六字节为时，第八字节为秒。
	year := binary.LittleEndian.Uint16(data[1:3])
	day := int(data[3])
	month := int(data[4])
	minute := int(data[5])
	hour := int(data[6])
	second := int(data[8])

	obj.reply.DateTime = time.Date(
		int(year),
		time.Month(month),
		day,
		hour,
		minute,
		second,
		0,
		time.Local,
	).Format("2006-01-02 15:04:05")
	// 以下文本段均为固定偏移定长字段，按 GBK 解码：名称第六十七至八十八字节，
	// 网址第八十九至一百五十二字节，类别第一百五十九至一百八十八字节。
	obj.reply.ServerName = Utf8ToGbk(data[67:89])
	obj.reply.Website = Utf8ToGbk(data[89:153])
	obj.reply.Category = Utf8ToGbk(data[159:189])
	obj.reply.Info = Utf8ToGbk(data[67:])
	return nil
}

// Response 返回解析完成后的首次握手响应结果。
func (obj *Hello1) Response() *Hello1Reply {
	return obj.reply
}
