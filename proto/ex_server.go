package proto

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

// ExLogin 表示扩展市场服务器登录握手协议，发送固定握手报文并解析服务器时间、名称与地址等信息。
type ExLogin struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExLoginReply
	contentHex string
}

// ExLoginReply 保存解码完成的登录握手响应结果。
type ExLoginReply struct {
	DateTime   string   // 服务器当前时间，格式为 `YYYY-MM-DD HH:MM:SS`，按本地时区组装。
	ServerName string   // 服务器名称，GBK 解码，取自偏移 61 起 21 字节。
	Desc       string   // 服务器描述信息，GBK 解码，取自偏移 93 起 151 字节。
	IP         string   // 服务器地址，GBK 解码，取自偏移 247 起 52 字节。
	Unknown    []string // 八个零散协议保留字段（`float32`//`uint16` 混合），按位宽读出后转字符串，语义待确认。
}

// NewExLogin 构造登录握手协议实例；该协议不接受入参，内置协议固定握手报文，返回的实例
// 零值不可直接使用。
func NewExLogin() *ExLogin {
	obj := &ExLogin{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExLoginReply),
		contentHex: "e5bb1c2fafe525941f32c6e5d53dfb415b734cc9cdbf0ac92021bfdd1eb06d22d008884c1611cb1378f6abd824d899d21f32c6e5d53dfb411f32c6e5d53dfb41a9325ac935dc0837335a16e4ce17c1bb",
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXLOGIN
	return obj
}

// BuildRequest 把内置固定握手报文解码为字节并编码为扩展协议请求字节流。
//
// 返回值第一项为编码后的请求字节流；内置十六进制握手报文格式非法时第二项返回非空错误。
func (obj *ExLogin) BuildRequest() ([]byte, error) {
	// 解码内置固定握手报文，十六进制格式非法时直接返回错误。
	payload, err := hex.DecodeString(obj.contentHex)
	if err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXLOGIN, payload)
}

// ParseResponse 从服务端响应字节流解码登录握手信息并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应固定区不足 299 字节时返回
// 非空错误，解码成功返回 `nil`。
func (obj *ExLogin) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 握手响应固定区至少需要 299 字节。
	if len(data) < 299 {
		return fmt.Errorf("invalid ex login response length: %d", len(data))
	}

	// 服务器时间各分量按固定偏移分别取出：年份为偏移 53 起 2 字节`uint16`，
	// 月份、日期、分钟、小时、秒依次为偏移 55、56、57、58、60 的。
	year := binary.LittleEndian.Uint16(data[53:55])
	month := int(data[55])
	day := int(data[56])
	minute := int(data[57])
	hour := int(data[58])
	second := int(data[60])

	obj.reply.DateTime = time.Date(int(year), time.Month(month), day, hour, minute, second, 0, time.Local).Format("2006-01-02 15:04:05")
	// 偏移 61:82 为服务器名称，93:244 为描述文本，247:299 为服务器地址，均为 GBK 编码。
	obj.reply.ServerName = Utf8ToGbk(data[61:82])
	obj.reply.Desc = Utf8ToGbk(data[93:244])
	obj.reply.IP = Utf8ToGbk(data[247:299])
	// 零散保留字段按各自位宽读出后格式化为字符串，语义待确认：82:86 为 `float32`，
	// 86、244、245、246 各占 1 字节，87:89、89:91、91:93 为 `uint16`。
	obj.reply.Unknown = []string{
		fmt.Sprintf("%g", float64(math.Float32frombits(binary.LittleEndian.Uint32(data[82:86])))),
		fmt.Sprintf("%d", data[86]),
		fmt.Sprintf("%d", binary.LittleEndian.Uint16(data[87:89])),
		fmt.Sprintf("%d", binary.LittleEndian.Uint16(data[89:91])),
		fmt.Sprintf("%d", binary.LittleEndian.Uint16(data[91:93])),
		fmt.Sprintf("%d", data[244]),
		fmt.Sprintf("%d", data[245]),
		fmt.Sprintf("%d", data[246]),
	}
	return nil
}

// Response 返回解码完成的登录握手响应对象。
func (obj *ExLogin) Response() *ExLoginReply {
	return obj.reply
}

// ExServerInfo 表示扩展市场服务器信息查询协议，返回网络延迟、版本号、签名与当前时间等信息；
// 该协议无请求参数体。
type ExServerInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExServerInfoReply
}

// ExServerInfoReply 保存解码完成的服务器信息响应结果。
type ExServerInfoReply struct {
	Delay       uint32 // 网络延迟，`uint32`，单位毫秒系推断，语义待确认。
	Info        string // 服务器附加信息，GBK 解码，取自偏移 16 起 25 字节。
	Version     string // 服务器版本号，GBK 解码，取自偏移 41 起 29 字节。
	ServerSign  string // 服务器签名，GBK 解码，取自偏移 117 起 13 字节。
	ServerSign2 string // 第二服务器签名，GBK 解码，取自偏移 240 起 13 字节，语义待确认。
	TimeNow     string // 服务器当前时间，格式为 `YYYY-MM-DD HH:MM:SS`，按本地时区组装。
	ServerName  string // 服务器名称，与 `Name` 同源。
	Name        string // 服务器显示名称，GBK 解码，取自偏移 159 起 30 字节。
}

// NewExServerInfo 构造服务器信息查询协议实例；该协议不接受入参，返回的实例零值不可直接使用。
func NewExServerInfo() *ExServerInfo {
	obj := &ExServerInfo{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExServerInfoReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXSERVERINFO
	return obj
}

// BuildRequest 构造服务器信息查询请求字节流。
//
// 该协议无载荷，仅由协议头与消息号组成；返回值第一项为请求字节流，不返回错误。
func (obj *ExServerInfo) BuildRequest() ([]byte, error) {
	return buildExRequest(KMSG_EXSERVERINFO, nil)
}

// ParseResponse 从服务端响应字节流解码服务器信息并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应固定区不足 327 字节时返回
// 非空错误，解码成功返回 `nil`。
func (obj *ExServerInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 服务器信息固定区至少需要 327 字节。
	if len(data) < 327 {
		return fmt.Errorf("invalid ex server info response length: %d", len(data))
	}

	// 偏移 0:4 为延迟`uint32`；16:41、41:70、117:130、240:253、159:189 为各段 GBK 文本。
	obj.reply.Delay = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Info = Utf8ToGbk(data[16:41])
	obj.reply.Version = Utf8ToGbk(data[41:70])
	obj.reply.ServerSign = Utf8ToGbk(data[117:130])
	obj.reply.ServerSign2 = Utf8ToGbk(data[240:253])

	// 偏移 80:84 为日期编号 `YYYYMMDD`，84:88 为时间编号 `HHMMSS`，均为`uint32`，
	// 分别拆分出年月日与时分秒后按本地时区组装。
	dateNow := binary.LittleEndian.Uint32(data[80:84])
	timeNow := binary.LittleEndian.Uint32(data[84:88])
	year := int(dateNow / 10000)
	month := int((dateNow % 10000) / 100)
	day := int(dateNow % 100)
	hour := int(timeNow / 10000)
	minute := int((timeNow % 10000) / 100)
	second := int(timeNow % 100)
	obj.reply.TimeNow = time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local).Format("2006-01-02 15:04:05")
	obj.reply.Name = Utf8ToGbk(data[159:189])
	// 服务器名称与显示名称取同一段文本。
	obj.reply.ServerName = obj.reply.Name
	return nil
}

// Response 返回解码完成的服务器信息响应对象。
func (obj *ExServerInfo) Response() *ExServerInfoReply {
	return obj.reply
}
