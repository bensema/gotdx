package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
)

// ExchangeAnnouncement 表示交易所公告协议，负责构造请求报文、解析响应并对外提供公告结果。
//
// 该协议请求只发送报文头，不携带业务负载；响应负载为空时解析返回错误。
// 协议对象内部保存已解析结果，非并发安全，不应在多个协程间共享同一实例。
type ExchangeAnnouncement struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExchangeAnnouncementReply
}

// ExchangeAnnouncementReply 保存交易所公告解析结果，包含公告版本号与解码后的公告正文。
type ExchangeAnnouncementReply struct {
	Version uint8  // 公告版本号，由服务端给定。
	Content string // 公告正文，已按 GBK 解码为 UTF-8，取自版本号之后的全部负载。
}

// NewExchangeAnnouncement 创建交易所公告协议对象，并初始化默认请求头；无入参，
// 返回已就绪的协议对象，对象内部状态非并发安全。
func NewExchangeAnnouncement() *ExchangeAnnouncement {
	obj := &ExchangeAnnouncement{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExchangeAnnouncementReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXCHANGEANNOUNCE
	return obj
}

// BuildRequest 把请求对象当前状态编码为协议字节流；编码失败时返回非空错误。
//
// 该协议无业务负载，报文体长度固定为两字节，两个长度字段互为冗余校验。
func (obj *ExchangeAnnouncement) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2
	obj.reqHeader.PkgLen2 = 2
	buf := new(bytes.Buffer)
	// 请求头整体写入，其后不追加业务负载。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的公告响应负载并填充内部响应结果；负载为空时返回错误。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节流；解析成功返回 `nil`，
// 负载长度非法时返回非空错误。
func (obj *ExchangeAnnouncement) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) == 0 {
		return fmt.Errorf("invalid exchange announcement response length: %d", len(data))
	}
	// 首字节为版本号，其后全部字节为公告正文。
	obj.reply.Version = data[0]
	obj.reply.Content = Utf8ToGbk(data[1:])
	return nil
}

// Response 返回解析完成后的交易所公告响应结果。
func (obj *ExchangeAnnouncement) Response() *ExchangeAnnouncementReply {
	return obj.reply
}

// HeartBeat 表示心跳保活协议，客户端定期发送以维持长连接，响应中带回服务端日期。
//
// 协议对象内部保存已解析结果，非并发安全；该协议请求只发送报文头，不携带业务负载。
type HeartBeat struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *HeartBeatReply
}

// HeartBeatReply 保存心跳响应结果，记录服务端返回的打包日期整数。
type HeartBeatReply struct {
	Date uint32 // 服务端打包日期，年月日依次占两位的四字节整数，如 20250512 对应 2025 年 5 月 12 日。
}

// NewHeartBeat 创建心跳协议对象，并初始化默认请求头；无入参，返回已就绪的协议对象。
func NewHeartBeat() *HeartBeat {
	obj := &HeartBeat{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(HeartBeatReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_HEARTBEAT
	return obj
}

// BuildRequest 把请求对象当前状态编码为心跳协议字节流；编码失败时返回非空错误。
//
// 心跳协议无业务负载，报文体长度固定为两字节，两个长度字段互为冗余校验。
func (obj *HeartBeat) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2
	obj.reqHeader.PkgLen2 = 2
	buf := new(bytes.Buffer)
	// 请求头整体写入，其后不追加业务负载。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	return buf.Bytes(), err
}

// ParseResponse 解析心跳响应负载；响应不足定长下限十个字节时返回错误，避免越界读取日期字段。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节流；解析成功返回 `nil`，
// 负载长度不足时返回非空错误。
func (obj *HeartBeat) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 10 {
		return fmt.Errorf("invalid heartbeat response length: %d", len(data))
	}
	// 第六至九字节为四字节打包日期，前六个字节的协议用途语义待确认。
	obj.reply.Date = binary.LittleEndian.Uint32(data[6:10])
	return nil
}

// Response 返回解析完成后的心跳响应结果。
func (obj *HeartBeat) Response() *HeartBeatReply {
	return obj.reply
}

// Announcement 表示服务商公告协议，响应由状态字节、失效日期与三段长度前缀文本组成，依次为标题、作者、正文。
//
// 协议对象内部保存已解析结果，非并发安全；服务端状态字节表示不存在公告时，响应结果保持空值。
type Announcement struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *AnnouncementReply
}

// AnnouncementReply 保存服务商公告解析结果，标记是否存在公告，并记录失效日期、标题、作者与正文。
type AnnouncementReply struct {
	HasContent bool   // 是否存在公告内容；服务端状态字节非一时保持 `false`。
	ExpireDate string // 公告失效日期，格式为年-月-日。
	Title      string // 公告标题，长度前缀文本段，已按 GBK 解码。
	Author     string // 公告作者，长度前缀文本段，已按 GBK 解码。
	Content    string // 公告正文，长度前缀文本段，已按 GBK 解码。
}

// NewAnnouncement 创建服务商公告协议对象，并初始化默认请求头；无入参，返回已就绪的协议对象。
func NewAnnouncement() *Announcement {
	obj := &Announcement{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(AnnouncementReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_ANNOUNCEMENT
	return obj
}

// BuildRequest 把请求头与协议要求的定长零填充负载编码为公告协议字节流；编码失败时返回非空错误。
//
// 报文体总长五十六字节：两字节头部长度加五十四字节零填充负载，两个长度字段互为冗余校验。
func (obj *Announcement) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 56
	obj.reqHeader.PkgLen2 = 56
	buf := new(bytes.Buffer)
	// 请求头整体写入。
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	// 追加五十四字节零填充，凑足协议要求的定长负载。
	_, err := buf.Write(make([]byte, 54))
	return buf.Bytes(), err
}

// ParseResponse 解析公告响应负载，按长度前缀依次取出标题、作者与正文；报文越界或长度非法时返回错误。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节流；解析成功返回 `nil`，
// 负载为空、长度越界或文本长度非法时返回非空错误。
func (obj *Announcement) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) == 0 {
		return fmt.Errorf("invalid announcement response length: %d", len(data))
	}
	// 状态字节非一表示服务端没有公告内容，保留空结果后直接返回。
	if data[0] != 0x01 {
		return nil
	}
	if len(data) < 11 {
		return fmt.Errorf("invalid announcement payload length: %d", len(data))
	}
	// 第一至四字节为打包日期，第五至十字节依次为标题、作者、正文的两字节长度。
	dateRaw := binary.LittleEndian.Uint32(data[1:5])
	titleLen := int(binary.LittleEndian.Uint16(data[5:7]))
	authorLen := int(binary.LittleEndian.Uint16(data[7:9]))
	contentLen := int(binary.LittleEndian.Uint16(data[9:11]))
	// 三段长度之和越过响应尾部即属于非法报文，先拦截再切片以防越界。
	end := 11 + titleLen + authorLen + contentLen
	if end > len(data) {
		return fmt.Errorf("invalid announcement text length: %d", len(data))
	}
	obj.reply.HasContent = true
	obj.reply.ExpireDate = fmt.Sprintf("%04d-%02d-%02d", dateRaw/10000, (dateRaw%10000)/100, dateRaw%100)
	pos := 11
	obj.reply.Title = Utf8ToGbk(data[pos : pos+titleLen])
	pos += titleLen
	obj.reply.Author = Utf8ToGbk(data[pos : pos+authorLen])
	pos += authorLen
	obj.reply.Content = Utf8ToGbk(data[pos : pos+contentLen])
	return nil
}

// Response 返回解析完成后的服务商公告响应结果。
func (obj *Announcement) Response() *AnnouncementReply {
	return obj.reply
}

// Info 表示服务端信息探测协议，用于获取网络时延、服务端标识、区域与当前时间等信息。
//
// 响应为定长大报文，各字段按固定字节偏移直接切片解析，不使用长度前缀；协议对象内部保存
// 已解析结果，非并发安全。响应不足定长下限时解析返回错误。
type Info struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *InfoReply
}

// InfoReply 保存服务端信息探测响应结果；标注为待确认的字段按原始数值与十六进制片段原样保留，不做语义解读。
type InfoReply struct {
	Delay       uint32    // 网络时延，四字节整数，计量单位语义待确认。
	Info        string    // 服务端信息文案，定长文本段，已按 GBK 解码。
	Content     string    // 公告正文，定长文本段，已按 GBK 解码。
	ServerSign  string    // 服务端签名，定长文本段，已按 GBK 解码。
	TimeNow     string    // 服务端当前时间，格式为年-月-日 时:分:秒；日期字段为零时保持空串。
	Unknown1    [3]string // 语义待确认，保留两个原始数值与一段十六进制片段。
	Unknown2    [3]string // 语义待确认，保留两个原始数值与一段十六进制片段。
	Unknown3    [3]uint16 // 语义待确认，由区域编号、待确认数值与开关位顺序组成。
	Region      uint16    // 区域编号，两字节整数。
	MaybeSwitch uint16    // 语义待确认的开关位，两字节整数。
}

// NewInfo 创建服务端信息探测协议对象，并初始化默认请求头；无入参，返回已就绪的协议对象。
func NewInfo() *Info {
	obj := &Info{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(InfoReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_PING
	return obj
}

// BuildRequest 把请求对象当前状态编码为信息探测协议字节流；编码失败时返回非空错误。
//
// 该协议无业务负载，报文体长度固定为两字节，两个长度字段互为冗余校验。
func (obj *Info) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2
	obj.reqHeader.PkgLen2 = 2
	buf := new(bytes.Buffer)
	// 请求头整体写入，其后不追加业务负载。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	return buf.Bytes(), err
}

// ParseResponse 按固定字节偏移解析服务端信息响应；响应不足定长下限四百二十七字节时返回错误。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节流；解析成功返回 `nil`，
// 负载长度不足时返回非空错误。文本段统一按 GBK 解码，日期与时间分别按年月日、时分秒打包为整数。
func (obj *Info) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 427 {
		return fmt.Errorf("invalid info response length: %d", len(data))
	}

	// 固定偏移依次为：时延第零至三字节、信息文案第十六至七十字节、公告正文第八十一至三百三十五字节、服务端签名第三百三十六至三百五十五字节。
	obj.reply.Delay = binary.LittleEndian.Uint32(data[0:4])
	obj.reply.Info = Utf8ToGbk(data[16:71])
	obj.reply.Content = Utf8ToGbk(data[81:336])
	obj.reply.ServerSign = Utf8ToGbk(data[336:356])
	// 以下偏移对应的字段协议语义待确认，先按原始数值与十六进制片段保存。
	unknown1A := binary.LittleEndian.Uint16(data[4:6])
	unknown1B := binary.LittleEndian.Uint16(data[14:16])
	unknown2A := binary.LittleEndian.Uint16(data[360:362])
	unknown2B := binary.LittleEndian.Uint16(data[362:364])
	unknown3B := binary.LittleEndian.Uint16(data[391:393])
	// 第三百九十七至四百字节为打包日期，第四百零一至四百零四字节为打包时间；日期为零时不填充时间。
	dateNow := binary.LittleEndian.Uint32(data[397:401])
	timeNow := binary.LittleEndian.Uint32(data[401:405])
	if dateNow != 0 {
		ts := time.Date(
			int(dateNow/10000),
			time.Month((dateNow%10000)/100),
			int(dateNow%100),
			int(timeNow/10000),
			int((timeNow%10000)/100),
			int(timeNow%100),
			0,
			time.Local,
		)
		obj.reply.TimeNow = ts.Format("2006-01-02 15:04:05")
	}
	// 区域编号第三百八十九至三百九十字节，待确认开关位第三百九十五至三百九十六字节。
	obj.reply.Region = binary.LittleEndian.Uint16(data[389:391])
	obj.reply.MaybeSwitch = binary.LittleEndian.Uint16(data[395:397])
	obj.reply.Unknown1 = [3]string{fmt.Sprintf("%d", unknown1A), fmt.Sprintf("%d", unknown1B), hex.EncodeToString(data[6:14])}
	obj.reply.Unknown2 = [3]string{fmt.Sprintf("%d", unknown2A), fmt.Sprintf("%d", unknown2B), hex.EncodeToString(data[364:370])}
	obj.reply.Unknown3 = [3]uint16{obj.reply.Region, unknown3B, obj.reply.MaybeSwitch}
	return nil
}

// Response 返回解析完成后的服务端信息探测响应结果。
func (obj *Info) Response() *InfoReply {
	return obj.reply
}
