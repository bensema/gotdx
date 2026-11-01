package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACSymbolBars 查询单只标的的 K 线序列与汇总行情。
//
// 使用时先由 NewMACSymbolBars 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACSymbolBars struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACSymbolBarsRequest
	reply      *MACSymbolBarsReply
}

// MACSymbolBarsRequest 保存 K 线查询的市场、代码、周期、起始位置、条数与复权标志参数。
type MACSymbolBarsRequest struct {
	Market   uint16   // 市场代码，两字节，取值见 `types.Market*` 常量。
	Code     [22]byte // 证券代码，GBK 编码定长 22 字节，不足补空字符。
	Period   uint16   // K 线周期类别，取值见 `types.KLINE_TYPE_*` 常量。
	Times    uint16   // 周期倍数，为 0 时默认取 1。
	Start    uint32   // 起始偏移条数，四字节。
	Count    uint16   // 请求条数；构造时内部加一，多取一条用于推算首根昨收基准。
	Adjust   uint16   // 复权方式，取值为 `types.AdjustNone`、`types.AdjustQFQ`、`types.AdjustHFQ`。
	Flag1    int8     // 协议标志位一，默认回填 1，具体含义语义待确认。
	Flag2    int8     // 协议标志位二，默认回填 1，具体含义语义待确认。
	Flag3    int8     // 协议标志位三，保留位，语义待确认。
	Flag4    int8     // 协议标志位四，默认回填 1，具体含义语义待确认。
	Zero     uint16   // 协议保留占位字段，语义待确认。
	Reserved [4]byte  // 协议保留填充字节，填零。
}

// MACSymbolBarsReply 保存 K 线响应的标的标识、K 线序列与汇总行情数据。
type MACSymbolBarsReply struct {
	Market       uint16         // 市场代码，两字节。
	Code         string         // 证券代码，GBK 解码。
	Period       uint8          // 周期类别回显字节。
	Unknown      uint16         // 协议保留槽位，语义待确认。
	Count        uint16         // 解析后 K 线条数，已扣除被舍弃的首条基准。
	Start        uint32         // 起始偏移回显，四字节。
	List         []MACSymbolBar // K 线序列，按时间顺序排列。
	Name         string         // 证券名称，GBK 解码。
	Decimal      uint8          // 价格小数位数。
	Category     uint16         // 证券类别代码，两字节。
	VolUnit      float64        // 成交量量纲换算系数，单精度浮点还原。
	DateTime     time.Time      // 汇总行情对应的本地时间。
	PreClose     float64        // 昨收价，单精度浮点还原。
	Open         float64        // 开盘价，单精度浮点还原。
	High         float64        // 最高价，单精度浮点还原。
	Low          float64        // 最低价，单精度浮点还原。
	Close        float64        // 收盘价，单精度浮点还原。
	Momentum     float64        // 动量指标，单精度浮点还原，语义待确认。
	Vol          uint32         // 汇总成交量，四字节。
	Amount       float64        // 汇总成交额，单精度浮点还原。
	Turnover     float64        // 换手率，单精度浮点还原。
	Avg          float64        // 均价，单精度浮点还原。
	Industry     uint32         // 行业分类编号，四字节。
	IndustryCode string         // 由行业编号换算得到的行业板块代码。
}

// MACSymbolBar 表示一根 K 线，包含开高低收、量额、流通股本、昨收基准与涨跌信息。
type MACSymbolBar struct {
	DateTime    time.Time // 组合后的本地时间。
	RawYMD      uint32    // 原始交易日打包字段（YYYYMMDD），未经时间修正。
	RawSeconds  uint32    // 原始当日秒数字段，未经时间修正。
	Open        float64   // 开盘价，单精度浮点还原。
	High        float64   // 最高价，单精度浮点还原。
	Low         float64   // 最低价，单精度浮点还原。
	Close       float64   // 收盘价，单精度浮点还原。
	Amount      float64   // 成交额，单精度浮点还原。
	Vol         float64   // 成交量，单精度浮点还原。
	FloatShares float64   // 流通股本，单精度浮点还原。
	Turnover    float64   // 换手率；条目记录未直接填充该槽位，口径语义待确认。
	PreClose    float64   // 昨收价，取上一根收盘价作为基准。
	RisePrice   float64   // 涨跌价，由收盘价相对昨收价计算。
	RiseRate    float64   // 涨跌幅，百分比数值。
}

// NewMACSymbolBars 创建 K 线查询对象；请求参数为空时使用内置默认参数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`Market` 为市场代码、`Code` 为证券代码、`Period` 为周期类别、
// `Times` 为周期倍数、`Start` 为起始偏移、`Count` 为请求条数、`Adjust` 为复权方式；
// `Times`、`Flag1`、`Flag2`、`Flag4` 为 0 时分别回填默认值 1。
func NewMACSymbolBars(req *MACSymbolBarsRequest) *MACSymbolBars {
	obj := &MACSymbolBars{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACSymbolBarsRequest),
		reply:      new(MACSymbolBarsReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACSYMBOLBARS
	obj.request.Times = 1
	obj.request.Flag1 = 1
	obj.request.Flag2 = 1
	obj.request.Flag4 = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACSymbolBars) applyRequest(req *MACSymbolBarsRequest) {
	if req.Times == 0 {
		req.Times = 1
	}
	if req.Flag1 == 0 {
		req.Flag1 = 1
	}
	if req.Flag2 == 0 {
		req.Flag2 = 1
	}
	if req.Flag4 == 0 {
		req.Flag4 = 1
	}
	// 请求条数加一：多取一条作为首根 K 线的昨收基准，解析阶段再舍弃该首条。
	req.Count = req.Count + 1
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACSYMBOLBARS` 与请求头封装成完整报文。
func (obj *MACSymbolBars) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACSYMBOLBARS, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或 K 线记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACSymbolBars) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 33 {
		return fmt.Errorf("invalid mac symbol bars response length: %d", len(data))
	}

	// 头部：市场（0:2）、GBK 代码（2:14）、周期（24）、计数与起始偏移（27:33）。
	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:14])
	obj.reply.Period = data[24]
	obj.reply.Unknown = binary.LittleEndian.Uint16(data[25:27])
	obj.reply.Count = binary.LittleEndian.Uint16(data[27:29])
	obj.reply.Start = binary.LittleEndian.Uint32(data[29:33])

	pos := 33
	var preCloseRaw float64 // 滚动保存上一根收盘价，作为当前根的昨收价
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+36 > len(data) {
			return fmt.Errorf("invalid mac symbol bar item %d", i)
		}
		// K 线记录布局：打包日期（0:4）、当日秒数（4:8），其后八段单精度浮点依次为
		// 开、高、低、收、成交额、成交量、流通股本。
		ymd := binary.LittleEndian.Uint32(data[pos : pos+4])
		seconds := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		item := MACSymbolBar{
			DateTime:    combineMACDateTime(ymd, seconds),
			RawYMD:      ymd,
			RawSeconds:  seconds,
			Open:        float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))),
			High:        float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))),
			Low:         float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+16 : pos+20]))),
			Close:       float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+20 : pos+24]))),
			Amount:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+24 : pos+28]))),
			Vol:         float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+28 : pos+32]))),
			FloatShares: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+32 : pos+36]))),
		}
		// 昨收与前收取上一根收盘价，随后把基准滚动为当前收盘价，再计算涨跌价与涨跌幅。
		item.PreClose = preCloseRaw
		preCloseRaw = item.Close
		item.RiseRate = item.GetRiseRate()
		item.RisePrice = item.GetRisePrice()
		pos += 36
		// 首条仅用于初始化昨收基准，不进入结果列表。
		if i == 0 {
			continue
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	// 结果条数比报文条数少一，即被舍弃的首条基准。
	obj.reply.Count = obj.reply.Count - 1

	// 汇总行情段不足 120 字节时直接返回，不再解析尾部字段。
	if pos+120 > len(data) {
		return nil
	}

	// 汇总段布局与行情快照尾部一致：名称、精度、类别、量纲、日期时间、开高低收、量额、换手、均价、行业。
	obj.reply.Name = Utf8ToGbk(data[pos : pos+44])
	obj.reply.Decimal = data[pos+44]
	obj.reply.Category = binary.LittleEndian.Uint16(data[pos+45 : pos+47])
	obj.reply.VolUnit = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+47 : pos+51])))
	dateRaw := binary.LittleEndian.Uint32(data[pos+56 : pos+60])
	timeRaw := binary.LittleEndian.Uint32(data[pos+60 : pos+64])
	obj.reply.DateTime = formatMACQuoteDateTime(dateRaw, timeRaw)
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+64 : pos+68])))
	obj.reply.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+68 : pos+72])))
	obj.reply.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+72 : pos+76])))
	obj.reply.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+76 : pos+80])))
	obj.reply.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+80 : pos+84])))
	obj.reply.Momentum = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+84 : pos+88])))
	obj.reply.Vol = binary.LittleEndian.Uint32(data[pos+88 : pos+92])
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+92 : pos+96])))
	obj.reply.Turnover = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+108 : pos+112])))
	obj.reply.Avg = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+112 : pos+116])))
	obj.reply.Industry = binary.LittleEndian.Uint32(data[pos+116 : pos+120])
	obj.reply.IndustryCode = macIndustryBoardSymbol(obj.reply.Industry)

	return nil
}

// GetRisePrice 返回收盘价相对昨收价的涨跌价；昨收价为 0 时改用开盘价作基准，避免差值失真。
func (bar MACSymbolBar) GetRisePrice() float64 {
	if bar.PreClose == 0 {
		// 昨收缺失时退用开盘价，避免差值被放大为接近收盘价本身。
		return bar.Close - bar.Open
	}
	return bar.Close - bar.PreClose
}

// GetRiseRate 返回以昨收价为基准的涨跌幅百分比；昨收价为 0 时改用开盘价作基准。
func (bar MACSymbolBar) GetRiseRate() float64 {
	if bar.PreClose == 0 {
		return (bar.Close - bar.Open) / (bar.Open) * 100
	}
	return (bar.Close - bar.PreClose) / (bar.PreClose) * 100
}

// Response 返回解码完成的 K 线响应；未调用 ParseResponse 前内容为零值。
func (obj *MACSymbolBars) Response() *MACSymbolBarsReply {
	return obj.reply
}

// combineMACDateTime 组合打包交易日与当日秒数为本地时间，不另行推断自然日期。
//
// 参数 `ymd` 为打包交易日（YYYYMMDD），`seconds` 为当日秒数；返回值为本地时区组合时间。
func combineMACDateTime(ymd uint32, seconds uint32) time.Time {
	year := int(ymd / 10000)
	month := int((ymd % 10000) / 100)
	day := int(ymd % 100)
	hours := int(seconds / 3600)
	minutes := int((seconds % 3600) / 60)

	return time.Date(year, time.Month(month), day, hours, minutes, 0, 0, time.Local)
}
