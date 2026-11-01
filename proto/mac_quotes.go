package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACQuotes 查询单只标的的实时行情快照与分时采样序列。
//
// 使用时先由 NewMACQuotes 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACQuotesRequest
	reply      *MACQuotesReply
}

// MACQuotesRequest 保存行情快照查询的市场、代码与查询日期参数，其余字段为协议保留占位。
type MACQuotesRequest struct {
	Market uint16   // 市场代码，两字节，取值见 `types.Market*` 常量。
	Code   [22]byte // 证券代码，GBK 编码定长 22 字节，不足补空字符。
	Zero1  uint16   // 查询日期低 16 位（`queryDate & 0xffff`），查询日期为 0 时表示最新交易日。
	Zero2  uint16   // 查询日期高 16 位（`queryDate >> 16`），与 `Zero1` 拼成完整日期整数。
	One    uint16   // 协议固定标志位，固定填 1，为 0 时由构造函数回填 1，语义待确认。
	Zero3  uint16   // 协议保留占位字段，语义待确认。
	Zero4  uint16   // 协议保留占位字段，语义待确认。
	Zero5  uint16   // 协议保留占位字段，语义待确认。
	Zero6  uint16   // 协议保留占位字段，语义待确认。
}

// MACQuotesReply 保存行情快照、分时采样序列与汇总行情数据。
type MACQuotesReply struct {
	Market       uint16              // 市场代码，两字节。
	Code         string              // 证券代码，GBK 解码。
	Date         uint32              // 行情日期打包整数（YYYYMMDD），四字节。
	Unknown      uint8               // 协议保留槽位，语义待确认。
	Price        float64             // 最新成交价，单精度浮点还原。
	Count        uint16              // 分时采样点数，两字节。
	ChartData    []MACQuoteChartItem // 分时采样序列，按时间顺序排列。
	Name         string              // 证券名称，GBK 解码。
	Decimal      uint8               // 价格小数位数。
	Category     uint16              // 证券类别代码，两字节。
	VolUnit      float64             // 成交量量纲换算系数，单精度浮点还原。
	DateTime     time.Time           // 汇总行情对应的本地时间。
	PreClose     float64             // 昨收价，单精度浮点还原。
	Open         float64             // 开盘价，单精度浮点还原。
	High         float64             // 最高价，单精度浮点还原。
	Low          float64             // 最低价，单精度浮点还原。
	Close        float64             // 收盘价，单精度浮点还原。
	Momentum     float64             // 动量指标，单精度浮点还原，语义待确认。
	Vol          uint32              // 累计成交量，四字节。
	Amount       float64             // 累计成交额，单精度浮点还原。
	Turnover     float64             // 换手率，单精度浮点还原。
	Avg          float64             // 均价，单精度浮点还原。
	Industry     uint32              // 行业分类编号，四字节。
	IndustryCode string              // 由行业编号换算得到的行业板块代码。
}

// MACQuoteChartItem 表示分时图上的一个采样点，包含分钟时间标签、成交价、均价、累计成交量与动量值。
type MACQuoteChartItem struct {
	Time     string  // 分时时间标签，格式 HH:MM:SS，由分钟偏移换算得到。
	Price    float64 // 该时点成交价，单精度浮点还原。
	Avg      float64 // 该时点均价，单精度浮点还原。
	Vol      uint32  // 累计成交量，四字节。
	Momentum float64 // 该时点动量值，单精度浮点还原，语义待确认。
}

// NewMACQuotes 创建行情快照查询对象；请求参数为空时使用内置默认参数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`Market` 为市场代码、`Code` 为证券代码；
// 查询日期由 `Zero1`、`Zero2` 分别保存低、高 16 位，传 0 表示取最新交易日；`One` 为 0 时回填 1。
func NewMACQuotes(req *MACQuotesRequest) *MACQuotes {
	obj := &MACQuotes{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACQuotesRequest),
		reply:      new(MACQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACQUOTES
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACQuotes) applyRequest(req *MACQuotesRequest) {
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACQUOTES` 与请求头封装成完整报文。
func (obj *MACQuotes) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACQUOTES, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACQuotes) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 35 {
		return fmt.Errorf("invalid mac quotes response length: %d", len(data))
	}

	// 35 字节头部：市场（0:2）、GBK 代码（2:24）、打包日期（24:28）、保留字节（28）、
	// 现价（29:33）、分时点数（33:35），整数、价格按单精度浮点还原。
	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:24])
	obj.reply.Date = binary.LittleEndian.Uint32(data[24:28])
	obj.reply.Unknown = data[28]
	obj.reply.Price = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[29:33])))
	obj.reply.Count = binary.LittleEndian.Uint16(data[33:35])

	// 分时点序列自偏移 35 开始，每条定长 18 字节。
	pos := 35
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+18 > len(data) {
			return fmt.Errorf("invalid mac quote chart item %d", i)
		}
		// 条目布局：分钟偏移（0:2）、价格（2:6）、均价（6:10）、累计成交量（10:14）、动量（14:18）。
		minutes := binary.LittleEndian.Uint16(data[pos : pos+2])
		item := MACQuoteChartItem{
			// 分钟偏移为分钟开始时刻，分时标签取分钟结束时点（整体后移一分钟），
			// 对齐行情展示口径：第一根 09:31、最后一根 15:00（与 `MACTickChart` 一致）。
			Time:     time.Date(0, 1, 1, int(minutes/60)%24, int(minutes%60), 0, 0, time.Local).Add(time.Minute).Format("15:04:05"),
			Price:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))),
			Avg:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+6 : pos+10]))),
			Vol:      binary.LittleEndian.Uint32(data[pos+10 : pos+14]),
			Momentum: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+14 : pos+18]))),
		}
		obj.reply.ChartData = append(obj.reply.ChartData, item)
		pos += 18
	}

	// 汇总行情段至少需要 109 字节，长度不足则报错，防止后续固定偏移切片越界。
	if pos+109 > len(data) {
		return fmt.Errorf("invalid mac quotes summary length: %d", len(data)-pos)
	}

	// 汇总段起始：GBK 名称（0:44）、精度字节（44）、类别（45:47）、量纲（47:51）。
	obj.reply.Name = Utf8ToGbk(data[pos : pos+44])
	obj.reply.Decimal = data[pos+44]
	obj.reply.Category = binary.LittleEndian.Uint16(data[pos+45 : pos+47])
	obj.reply.VolUnit = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+47 : pos+51])))
	// 日期（56:60）与时间（60:64）为两个打包整数，分别按年月日、时分秒拆分。
	dateRaw := binary.LittleEndian.Uint32(data[pos+56 : pos+60])
	timeRaw := binary.LittleEndian.Uint32(data[pos+60 : pos+64])
	obj.reply.DateTime = formatMACQuoteDateTime(dateRaw, timeRaw)
	// 偏移 64 到 84 依次为昨收、开、高、低、收，均为单精度浮点。
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+64 : pos+68])))
	obj.reply.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+68 : pos+72])))
	obj.reply.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+72 : pos+76])))
	obj.reply.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+76 : pos+80])))
	obj.reply.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+80 : pos+84])))
	obj.reply.Momentum = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+84 : pos+88])))
	// 88:92 成交量、92:96 成交额；108:112 换手率、112:116 均价；116:120 行业编号。
	obj.reply.Vol = binary.LittleEndian.Uint32(data[pos+88 : pos+92])
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+92 : pos+96])))
	obj.reply.Turnover = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+108 : pos+112])))
	obj.reply.Avg = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+112 : pos+116])))
	obj.reply.Industry = binary.LittleEndian.Uint32(data[pos+116 : pos+120])
	// 行业编号按前缀规则换算为行业板块代码字符串。
	obj.reply.IndustryCode = macIndustryBoardSymbol(obj.reply.Industry)

	return nil
}

// Response 返回解码完成的行情快照响应；未调用 ParseResponse 前内容为零值。
func (obj *MACQuotes) Response() *MACQuotesReply {
	return obj.reply
}

// formatMACQuoteDateTime 将打包日期与打包时间两个整数拆分，组合为本地时间。
//
// 参数 `dateRaw` 为打包日期整数（YYYYMMDD），`timeRaw` 为打包时间整数（HHMMSS）；
// 返回值为本地时区组合后的时间。
func formatMACQuoteDateTime(dateRaw uint32, timeRaw uint32) time.Time {
	year := int(dateRaw / 10000)
	month := int((dateRaw % 10000) / 100)
	day := int(dateRaw % 100)
	hour := int(timeRaw / 10000)
	minute := int((timeRaw % 10000) / 100)
	second := int(timeRaw % 100)
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
}
