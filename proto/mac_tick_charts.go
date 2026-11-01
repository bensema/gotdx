package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACTickCharts 查询单只标的的多日分时序列与汇总行情。
//
// 使用时先由 NewMACTickCharts 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACTickCharts struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACTickChartsRequest
	reply      *MACTickChartsReply
}

// MACTickChartsRequest 保存多日分时查询的市场、代码、查询日期、交易日数与保留占位参数。
type MACTickChartsRequest struct {
	Market    uint16   // 市场代码，两字节，取值见 `types.Market*` 常量。
	Code      [22]byte // 证券代码，GBK 编码定长 22 字节，不足补空字符。
	QueryDate uint32   // 查询日期打包整数（YYYYMMDD），传 0 表示取最新交易日。
	Days      uint16   // 请求返回的交易日数，为 0 时默认取 5；服务端多余交易日会按此值裁剪。
	One       uint16   // 协议固定标志位，固定填 1，语义待确认。
	Reserved  [6]byte  // 协议保留填充字节，填零。
}

// MACTickChartsReply 保存多日分时响应的标的标识、交易日计数、逐日分时数据与汇总行情。
type MACTickChartsReply struct {
	Market       uint16            // 市场代码，两字节。
	Code         string            // 证券代码，GBK 解码。
	Count        uint16            // 服务端返回的交易日数，裁剪后与 `Charts` 长度一致。
	SendLast     uint8             // 末点标记字节，语义待确认。
	PageSize     uint16            // 分页大小，两字节，语义待确认。
	Total        uint16            // 分时点总数，两字节。
	Charts       []MACTickChartDay // 按交易日分组的分时点序列。
	Name         string            // 证券名称，GBK 解码。
	Decimal      uint8             // 价格小数位数。
	Category     uint16            // 证券类别代码，两字节。
	VolUnit      float64           // 成交量量纲换算系数，单精度浮点还原。
	DateTime     time.Time         // 汇总行情对应的本地时间。
	PreClose     float64           // 昨收价，单精度浮点还原。
	Open         float64           // 开盘价，单精度浮点还原。
	High         float64           // 最高价，单精度浮点还原。
	Low          float64           // 最低价，单精度浮点还原。
	Close        float64           // 收盘价，单精度浮点还原。
	Momentum     float64           // 动量指标，单精度浮点还原，语义待确认。
	Vol          uint32            // 汇总成交量，四字节。
	Amount       float64           // 汇总成交额，单精度浮点还原。
	Turnover     float64           // 换手率，单精度浮点还原。
	Avg          float64           // 均价，单精度浮点还原。
	Industry     uint32            // 行业分类编号，四字节。
	IndustryCode string            // 由行业编号换算得到的行业板块代码。
}

// MACTickChartDay 表示一个交易日的分时数据，包含日期、当日昨收与该日分时点序列。
type MACTickChartDay struct {
	Date     string             // 交易日文本，格式 YYYY-MM-DD；头部日期为零时留空。
	PreClose float64            // 当日昨收价，由头部昨收数组按交易日序号取出。
	Ticks    []MACTickChartItem // 该交易日的分时点序列，按时间顺序排列。
}

// MACTickChartItem 表示一个分时点，包含时间标签、价格、均价、成交量与派生成交额；
// 其中一个保留槽位语义待确认，原样保存。
type MACTickChartItem struct {
	Time    string  // 分时时间标签，格式 HH:MM:SS，由分钟偏移换算并整体后移一分钟。
	Price   float64 // 该时点成交价，单精度浮点还原。
	Avg     float64 // 该时点均价，单精度浮点还原。
	Vol     uint16  // 分时累计成交量，两字节。
	Amount  float64 // 派生成交额，由成交量、均价与固定系数 100 推算，报文未直接给出。
	Unknown uint16  // 协议保留槽位，语义待确认。
}

// NewMACTickCharts 创建多日分时查询对象；请求参数为空时使用内置默认交易日数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：其中 `Market` 为市场代码、`Code` 为证券代码、
// `QueryDate` 为查询日期（传 0 表示最新交易日）、`Days` 为请求交易日数；`Days` 为 0 时回填默认值 5。
func NewMACTickCharts(req *MACTickChartsRequest) *MACTickCharts {
	obj := &MACTickCharts{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACTickChartsRequest),
		reply:      new(MACTickChartsReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACTICKCHARTS
	obj.request.Days = 5
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACTickCharts) applyRequest(req *MACTickChartsRequest) {
	if req.Days == 0 {
		req.Days = 5
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACTICKCHARTS` 与请求头封装成完整报文。
func (obj *MACTickCharts) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACTICKCHARTS, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并按交易日分组填充响应；报文长度不足或记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
// 分组完成后按请求天数裁剪：服务端返回的交易日数多于请求天数时，仅保留最近 `Days` 个交易日，
// 并同步修正交易日计数，保证 `Charts` 长度与 `Count` 一致。
func (obj *MACTickCharts) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 71 {
		return fmt.Errorf("invalid mac tick charts response length: %d", len(data))
	}

	// 响应头：市场代码（0:2）、GBK 证券代码（2:24，定长 22 字节）。
	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:24])

	// 头部预留最近五个交易日：日期数组位于 24 到 44，昨收数组位于 44 到 64，各五个四字节条目。
	var dates [5]uint32
	var preCloses [5]float32
	for i := range dates {
		dates[i] = binary.LittleEndian.Uint32(data[24+i*4 : 28+i*4])
		preCloses[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[44+i*4 : 48+i*4]))
	}

	// 偏移 64 到 71 依次为交易日数、末点标记、分页大小、总点数。
	obj.reply.Count = binary.LittleEndian.Uint16(data[64:66])
	obj.reply.SendLast = data[66]
	obj.reply.PageSize = binary.LittleEndian.Uint16(data[67:69])
	obj.reply.Total = binary.LittleEndian.Uint16(data[69:71])

	pos := 71
	// 临时保留分钟偏移，供分组时判断是否跨日。
	type tickWithMinute struct {
		minute uint16
		item   MACTickChartItem
	}

	ticks := make([]tickWithMinute, 0, obj.reply.Total)
	for tickIndex := uint16(0); tickIndex < obj.reply.Total; tickIndex++ {
		if pos+14 > len(data) {
			return fmt.Errorf("invalid mac tick charts item %d", tickIndex)
		}
		// 分时点布局：分钟偏移（0:2）、价格（2:6）、均价（6:10）、成交量（10:12）、保留槽位（12:14）。
		minutes := binary.LittleEndian.Uint16(data[pos : pos+2])
		MACTickChartItemInfo := MACTickChartItem{
			// 分钟偏移换算为时间标签，再整体后移一分钟以对齐行情展示口径。
			Time:    time.Date(0, 1, 1, int(minutes/60)%24, int(minutes%60), 0, 0, time.Local).Add(time.Minute).Format("15:04:05"),
			Price:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))),
			Avg:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+6 : pos+10]))),
			Vol:     binary.LittleEndian.Uint16(data[pos+10 : pos+12]),
			Unknown: binary.LittleEndian.Uint16(data[pos+12 : pos+14]),
		}
		// 成交额由成交量、均价与固定系数 100 推算，报文未直接给出。
		MACTickChartItemInfo.Amount = float64(MACTickChartItemInfo.Vol) * MACTickChartItemInfo.Avg * 100
		ticks = append(ticks, tickWithMinute{
			minute: minutes,
			item:   MACTickChartItemInfo,
		})
		pos += 14
	}

	if obj.reply.Count == 0 {
		obj.reply.Charts = nil
	} else {
		currentDay := 0
		day := newMACTickChartDay(dates, preCloses, currentDay)
		for i, tick := range ticks {
			day.Ticks = append(day.Ticks, tick.item)
			// 最后一个交易日不再继续切分。
			if currentDay >= int(obj.reply.Count)-1 {
				continue
			}
			// 下一点分钟回绕到更小值，说明进入新交易日，封存当前日并开启新日。
			if i+1 < len(ticks) && ticks[i+1].minute <= tick.minute {
				obj.reply.Charts = append(obj.reply.Charts, day)
				currentDay++
				day = newMACTickChartDay(dates, preCloses, currentDay)
			}
		}
		obj.reply.Charts = append(obj.reply.Charts, day)
		// 交易日数多于实际分组时，补齐空交易日占位，使逐日序列长度与请求交易日数一致。
		for len(obj.reply.Charts) < int(obj.reply.Count) {
			obj.reply.Charts = append(obj.reply.Charts, newMACTickChartDay(dates, preCloses, len(obj.reply.Charts)))
		}
	}

	// 服务端返回的交易日数可能多于请求天数（如请求 1 天时服务端附送前一交易日作基准日），
	// 按请求天数裁剪，仅保留最近 `Days` 个交易日，并同步修正交易日计数，保证 `Charts` 与 `Count` 长度一致。
	if n := int(obj.request.Days); len(obj.reply.Charts) > n {
		obj.reply.Charts = obj.reply.Charts[:n]
		obj.reply.Count = uint16(n)
	}

	// 汇总行情段不足 120 字节时直接返回，跳过尾部字段解析。
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

// Response 返回解码完成的多日分时响应；未调用 ParseResponse 前内容为零值。
func (obj *MACTickCharts) Response() *MACTickChartsReply {
	return obj.reply
}

// formatMACDate 将打包日期整数拆出年、月、日，格式化为短横线分隔的日期文本，取本地时区零点。
//
// 参数 `raw` 为打包日期整数（YYYYMMDD）；返回值格式为 `2006-01-02`。
func formatMACDate(raw uint32) string {
	year := int(raw / 10000)
	month := int((raw % 10000) / 100)
	day := int(raw % 100)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
}

// newMACTickChartDay 按交易日序号从头部日期与昨收数组取出对应值，构造空的日分时容器；
// 序号越界或日期为零时保留空日期与零昨收。
//
// 参数 `dates` 为头部五个打包交易日，`preCloses` 为对应五个昨收单精度浮点，
// `dayIndex` 为目标交易日在数组中的序号；返回值仅含日期、昨收与空分时序列。
func newMACTickChartDay(dates [5]uint32, preCloses [5]float32, dayIndex int) MACTickChartDay {
	day := MACTickChartDay{}
	if dayIndex < len(dates) && dates[dayIndex] != 0 {
		day.Date = formatMACDate(dates[dayIndex])
		day.PreClose = float64(preCloses[dayIndex])
	}
	return day
}
