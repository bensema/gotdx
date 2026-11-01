package proto

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// MACServerInfo 查询服务端的交易日、交易时段、时间戳与市场参数等基础信息。
//
// 使用时先由 NewMACServerInfo 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACServerInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACServerInfoRequest
	reply      *MACServerInfoReply
}

// MACServerInfoRequest 保存服务端信息请求的固定 68 字节负载。
type MACServerInfoRequest struct {
	Payload [68]byte // 固定 68 字节请求负载，全零时使用内置默认负载。
}

// MACTradingSession 表示一段交易时段，以距午夜的分钟数记录开盘与收盘时点，并换算为可读时间文本。
type MACTradingSession struct {
	OpenMinutes  uint16 // 开盘距午夜的分钟数，两字节。
	CloseMinutes uint16 // 收盘距午夜的分钟数，两字节。
	Open         string // 开盘可读时间文本，格式 `H:MM`。
	Close        string // 收盘可读时间文本，格式 `H:MM`。
}

// MACServerInfoReply 保存服务端信息响应的计数、标志、日期、两组交易时段、时间戳、市场参数与备查十六进制段。
type MACServerInfoReply struct {
	Count           uint16              // 记录条数，两字节。
	FlagsHex        string              // 头部 8 字节标志位的十六进制文本，仅供备查。
	Tag             string              // 3 字节标签文本，尾部以空字符填充。
	Today           string              // 当日日期，格式 YYYY-MM-DD。
	TS1             uint32              // 当日时间戳，四字节，口径语义待确认。
	Sessions1       []MACTradingSession // 第一组交易时段，至多 4 段。
	Sessions2       []MACTradingSession // 第二组交易时段，至多 4 段。
	Flag            uint8               // 状态标志字节，取值含义语义待确认。
	LastTradingDay  string              // 上一交易日日期，格式 YYYY-MM-DD。
	TS2             uint32              // 上一交易日时间戳，四字节，口径语义待确认。
	LastTradingDay2 string              // 第二参考交易日日期，口径语义待确认。
	TS3             uint32              // 第二参考时间戳，四字节，口径语义待确认。
	MarketParam1    uint32              // 市场参数一，四字节，含义语义待确认。
	MarketParam2    uint32              // 市场参数二，四字节，含义语义待确认。
	ExtraHex        string              // 尾部剩余字节的十六进制文本，仅供备查。
}

// NewMACServerInfo 创建服务端信息查询对象；请求参数为空时使用内置默认负载。
//
// 参数 `req` 指定预填请求参数，可传 `nil`：`Payload` 为 68 字节固定负载；
// 负载全零时回填 `defaultMACServerInfoPayload` 返回的默认负载。
func NewMACServerInfo(req *MACServerInfoRequest) *MACServerInfo {
	obj := &MACServerInfo{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACServerInfoRequest),
		reply:      new(MACServerInfoReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACSERVERINFO
	obj.request.Payload = defaultMACServerInfoPayload()
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACServerInfo) applyRequest(req *MACServerInfoRequest) {
	if req.Payload == ([68]byte{}) {
		req.Payload = defaultMACServerInfoPayload()
	}
	obj.request = req
}

// defaultMACServerInfoPayload 返回服务端信息请求的固定 68 字节默认负载，
// 仅在协议约定的偏移处填入标志与日期样例字节，其余补零；返回值直接作为请求 `Payload` 使用。
func defaultMACServerInfoPayload() [68]byte {
	var payload [68]byte
	copy(payload[:4], []byte{0x04, 0x00, 0x2d, 0x31})
	copy(payload[12:16], []byte{0x00, 0x27, 0x06, 0x0e})
	return payload
}

// BuildRequest 把固定 68 字节负载封装为完整协议报文；负载长度固定，不存在序列化失败。
//
// 该方法无外部入参，直接取已填充的 `Payload`，由 `buildExRequest` 追加命令码
// `KMSG_MACSERVERINFO` 与请求头封装成完整报文。
func (obj *MACServerInfo) BuildRequest() ([]byte, error) {
	return buildExRequest(KMSG_MACSERVERINFO, obj.request.Payload[:])
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足最小定长要求时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACServerInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 最小定长 87 字节，不足即判定为非法报文，避免后续切片越界。
	if len(data) < 87 {
		return fmt.Errorf("invalid mac server info response length: %d", len(data))
	}

	// 先读取条数；随后 8 字节标志位整体转十六进制备查。
	pos := 0
	obj.reply.Count = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2
	obj.reply.FlagsHex = hex.EncodeToString(data[pos : pos+8])
	pos += 8
	// 3 字节标签以空字符填充；其后 9 字节为协议保留区，直接跳过。
	obj.reply.Tag = strings.TrimRight(string(data[pos:pos+3]), "\x00")
	pos += 3
	pos += 9

	// 当日日期与时间戳，随后两组各 16 字节交易时段。
	obj.reply.Today = formatMACServerDate(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	obj.reply.TS1 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	obj.reply.Sessions1 = parseMACTradingSessions(data[pos : pos+16])
	pos += 16
	obj.reply.Sessions2 = parseMACTradingSessions(data[pos : pos+16])
	pos += 16
	// 1 字节状态标志。
	obj.reply.Flag = data[pos]
	pos++
	// 上一交易日日期、时间戳、第二交易日日期、时间戳顺序排列。
	obj.reply.LastTradingDay = formatMACServerDate(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	obj.reply.TS2 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	obj.reply.LastTradingDay2 = formatMACServerDate(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	obj.reply.TS3 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	// 末尾两个市场参数；其后若仍有剩余字节，整体转十六进制备查。
	obj.reply.MarketParam1 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	obj.reply.MarketParam2 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	if pos < len(data) {
		obj.reply.ExtraHex = hex.EncodeToString(data[pos:])
	}
	return nil
}

// parseMACTradingSessions 把 16 字节时段缓冲按每 4 字节一组解析为开盘与收盘分钟数对，共至多 4 段；
// 每组两个两字节整数分别为开盘、收盘距午夜的分钟数，均读取。
//
// 参数 `data` 为一段交易时段缓冲（16 字节）；返回值为解析后的时段列表，顺序与缓冲内组序一致。
func parseMACTradingSessions(data []byte) []MACTradingSession {
	sessions := make([]MACTradingSession, 0, 4)
	for i := 0; i+4 <= len(data); i += 4 {
		openMinutes := binary.LittleEndian.Uint16(data[i : i+2])
		closeMinutes := binary.LittleEndian.Uint16(data[i+2 : i+4])
		sessions = append(sessions, MACTradingSession{
			OpenMinutes:  openMinutes,
			CloseMinutes: closeMinutes,
			Open:         formatMACSessionTime(openMinutes),
			Close:        formatMACSessionTime(closeMinutes),
		})
	}
	return sessions
}

// formatMACSessionTime 将距午夜的分钟数格式化为可读时分文本。
//
// 参数 `minutes` 为距午夜的分钟数；返回值格式为 `H:MM`。
func formatMACSessionTime(minutes uint16) string {
	return fmt.Sprintf("%d:%02d", minutes/60, minutes%60)
}

// formatMACServerDate 将打包日期整数拆出年、月、日并格式化为短横线分隔的日期文本。
//
// 参数 `raw` 为打包日期整数（YYYYMMDD）；返回值格式为 `YYYY-MM-DD`。
func formatMACServerDate(raw uint32) string {
	return fmt.Sprintf("%04d-%02d-%02d", raw/10000, (raw%10000)/100, raw%100)
}

// Response 返回解码完成的服务端信息响应；未调用 ParseResponse 前内容为零值。
func (obj *MACServerInfo) Response() *MACServerInfoReply {
	return obj.reply
}
