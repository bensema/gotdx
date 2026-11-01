package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

// GetQuotesList 表示一次按分类与排序条件获取行情列表协议的完整流程对象。
//
// 该对象持有请求头、请求参数与响应数据；零值不可直接使用，必须通过
// NewGetQuotesList 构造。对象绑定单次查询流程，不支持多协程并发复用。
type GetQuotesList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetQuotesListRequest
	reply      *GetQuotesListReply
}

// GetQuotesListRequest 是 GetQuotesList 查询使用的请求参数。
type GetQuotesListRequest struct {
	Category    uint16 // 板块或市场分类，`uint16`。
	SortType    uint16 // 排序方式，`uint16`，取值对应排序类别常量。
	Start       uint16 // 起始偏移，`uint16`，从 0 开始按页递增。
	Count       uint16 // 请求条数，`uint16`。
	SortReverse uint16 // 是否倒序排序，0 为正序、非 0 为倒序。
	Mode        uint16 // 协议模式位，`uint16`；为 0 时默认取 5。
	Filter      uint16 // 过滤条件，`uint16`。
	One         uint16 // 协议固定位，`uint16`；为 0 时默认取 1。
	Zero        uint16 // 协议保留占位，填 0。
}

// GetQuotesListReply 是 GetQuotesList 查询解析完成后的响应数据。
type GetQuotesListReply struct {
	Block uint16          // 区块或分组标识，2 字节`uint16`。
	Count uint16          // 返回条数，2 字节`uint16`。
	List  []QuoteListItem // 排序后的行情快照列表，长度与 `Count` 一致。
}

// QuoteListItem 表示列表中的一条行情快照，包含价格、量额、统计指标与买卖档位。
type QuoteListItem struct {
	Market        uint8   // 市场代码，1 字节无符号整数。
	Code          string  // 6 字节定长证券代码，按 `GBK` 还原。
	Active1       uint16  // 活跃度字段 1，2 字节`uint16`，语义待确认。
	Active2       uint16  // 活跃度字段 2，2 字节`uint16`，语义待确认。
	Close         float64 // 最新价，单位为元，由基准价直接还原。
	Price         float64 // 当前价，单位为元，通常与最新价一致。
	PreClose      float64 // 昨收价，单位为元，基准价加差值后除以 100。
	Open          float64 // 今开价，单位为元，基准价加差值后除以 100。
	High          float64 // 最高价，单位为元，基准价加差值后除以 100。
	Low           float64 // 最低价，单位为元，基准价加差值后除以 100。
	ServerTime    string  // 服务端时间，由变长整数经 `formatServerTime` 格式化。
	NegPrice      float64 // 特殊价格字段，基准差值除以 100，语义待确认。
	Vol           int     // 总成交量，变长整数解码。
	CurVol        int     // 现量（最近一笔成交量），变长整数解码。
	Amount        float64 // 总成交额，4 字节的单精度浮点数。
	InVol         int     // 内盘量（主动卖盘成交量），变长整数解码。
	OutVol        int     // 外盘量（主动买盘成交量），变长整数解码。
	SAmount       int     // 上涨家数或卖出相关统计字段，变长整数解码，语义待确认。
	OpenAmount    int     // 开盘金额，变长整数解码。
	BidLevels     []Level // 买盘一档档位，价格相对基准价偏移。
	AskLevels     []Level // 卖盘一档档位，价格相对基准价偏移。
	Unknown       uint16  // 尾部未知字段，2 字节`uint16`，语义待确认。
	RiseSpeed     float64 // 涨速，放大 100 倍的有符号短整数，解析后除以 100。
	ShortTurnover float64 // 短周期换手，放大 100 倍的有符号短整数，解析后除以 100。
	Min2Amount    float64 // 两分钟成交额，4 字节的单精度浮点数。
	OpeningRush   float64 // 开盘抢筹指标，放大 100 倍的有符号短整数，解析后除以 100。
	VolRiseSpeed  float64 // 量涨速，4 字节的单精度浮点数。
	Depth         float64 // 委比或深度指标，4 字节的单精度浮点数，语义待确认。
	Turnover      float64 // 换手率，由高层接口尽力估算补齐，协议本身不下发。
}

// NewGetQuotesList 构造排序行情列表查询对象。
//
// 参数 `req` 指定分类 `Category`、排序方式 `SortType`、起始偏移 `Start`、条数 `Count`
// 与倒序标记 `SortReverse`；`Mode` 为 0 时默认 5、`One` 为 0 时默认 1；`req` 为
// `nil` 时使用空请求参数。返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetQuotesList(req *GetQuotesListRequest) *GetQuotesList {
	obj := new(GetQuotesList)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetQuotesListRequest)
	obj.reply = new(GetQuotesListReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_QUOTESLIST
	obj.request.Mode = 5
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetQuotesList) applyRequest(req *GetQuotesListRequest) {
	if req.Mode == 0 {
		req.Mode = 5
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文。
//
// 返回的字节切片可直接写入连接；`PkgLen1` 与 `PkgLen2` 双写为同一负载长度，
// 缓冲区写入失败时返回错误。
func (obj *GetQuotesList) BuildRequest() ([]byte, error) {
	// 请求参数固定 20 字节，包头长度双写为 20。
	obj.reqHeader.PkgLen1 = 20
	obj.reqHeader.PkgLen2 = 20

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的排序行情列表并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；解码失败时返回错误。
func (obj *GetQuotesList) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 响应头部 4 字节：前 2 字节区块标识，后 2 字节条数。
	obj.reply.Block = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Count = binary.LittleEndian.Uint16(data[2:4])
	pos := 4

	for i := uint16(0); i < obj.reply.Count; i++ {
		// 共享解码器：返回解码后的条目与更新后的读取偏移。
		item, nextPos, err := parseQuoteListItem(data, pos)
		if err != nil {
			return err
		}
		pos = nextPos
		obj.reply.List = append(obj.reply.List, item)
	}

	return nil
}

// Response 返回已解析完成的排序行情列表响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetQuotesList) Response() *GetQuotesListReply {
	return obj.reply
}

// GetQuotes 表示一次按证券列表批量获取行情快照协议的完整流程对象。
//
// 该对象与 GetQuotesList 的响应结构和解码流程一致；区别仅在于请求侧不按分类
// 排序，而是直接传入一组市场与代码。零值不可直接使用，必须通过 NewGetQuotes
// 构造，对象绑定单次查询流程，不支持多协程并发复用。
type GetQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetQuotesRequest
	reply      *GetQuotesReply
}

// GetQuotesRequest 是 GetQuotes 批量行情查询使用的请求参数。
type GetQuotesRequest struct {
	Stocks []Stock // 待查询证券列表，每项由市场 `Market` 与代码 `Code` 唯一标识。
}

// GetQuotesReply 是 GetQuotes 批量行情查询解析完成后的响应数据。
type GetQuotesReply struct {
	Block uint16          // 区块或分组标识，2 字节`uint16`。
	Count uint16          // 返回条数，2 字节`uint16`。
	List  []QuoteListItem // 批量行情快照列表，长度与 `Count` 一致。
}

// NewGetQuotes 构造批量行情查询对象。
//
// 参数 `req` 指定待查询证券列表 `Stocks`；`req` 为 `nil` 时使用空请求参数。
// 返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetQuotes(req *GetQuotesRequest) *GetQuotes {
	obj := new(GetQuotes)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetQuotesRequest)
	obj.reply = new(GetQuotesReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_QUOTES
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetQuotes) applyRequest(req *GetQuotesRequest) {
	obj.request = req
}

// BuildRequest 将待查询证券列表编码为协议报文。
//
// 负载由固定模式字、6 字节保留区、证券个数与逐只市场/代码段组成；返回的字节切片
// 可直接写入连接，缓冲区写入失败时返回错误。
func (obj *GetQuotes) BuildRequest() ([]byte, error) {
	// 负载长度 = 个数 2 + 固定段 10 + 每只证券 7（市场 1 + 代码 6），包头长度双写。
	obj.reqHeader.PkgLen1 = 2 + 10 + uint16(len(obj.request.Stocks)*7)
	obj.reqHeader.PkgLen2 = 2 + 10 + uint16(len(obj.request.Stocks)*7)

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	// 固定模式字 5。
	if err := binary.Write(buf, binary.LittleEndian, uint16(5)); err != nil {
		return nil, err
	}
	// 6 字节协议保留区，整体补零。
	if _, err := buf.Write(make([]byte, 6)); err != nil {
		return nil, err
	}
	// 写入待查询证券个数。
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(obj.request.Stocks))); err != nil {
		return nil, err
	}
	for _, stock := range obj.request.Stocks {
		code := make([]byte, 6)
		copy(code, stock.Code)
		if err := binary.Write(buf, binary.LittleEndian, stock.Market); err != nil {
			return nil, err
		}
		if _, err := buf.Write(code); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// ParseResponse 解析服务端返回的批量行情列表并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；解码失败时返回错误。
func (obj *GetQuotes) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	obj.reply.Block = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Count = binary.LittleEndian.Uint16(data[2:4])
	pos := 4

	for i := uint16(0); i < obj.reply.Count; i++ {
		item, nextPos, err := parseQuoteListItem(data, pos)
		if err != nil {
			return err
		}
		pos = nextPos
		obj.reply.List = append(obj.reply.List, item)
	}

	return nil
}

// Response 返回已解析完成的批量行情响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetQuotes) Response() *GetQuotesReply {
	return obj.reply
}

// parseQuoteListItem 解码一条排序或批量行情快照，返回条目与下一条记录的读取偏移。
//
// 参数 `data` 为完整响应负载字节，`pos` 为本条记录的起始偏移；返回解码后的
// `QuoteListItem`、推进到下一条记录的偏移 `pos` 与解码错误。单条记录按固定顺序
// 解码：定长头部之后是基准价与各价格差值、量额字段与一档买卖，末尾为速度、换手等
// 统计字段与多段保留区；价格以基准价加差值还原后除以 100 得到元。
func parseQuoteListItem(data []byte, pos int) (QuoteListItem, int, error) {
	item := QuoteListItem{}

	// 市场 1 字节。
	item.Market = data[pos]
	pos++
	// 代码 6 字节，按 `GBK` 还原字符串。
	item.Code = Utf8ToGbk(data[pos : pos+6])
	pos += 6
	// 活跃度字段 1，2 字节。
	item.Active1 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2

	// 基准价与各价格差值；后续价位为基准价加差值。
	basePrice := getprice(data, &pos)
	preCloseDiff := getprice(data, &pos)
	openDiff := getprice(data, &pos)
	highDiff := getprice(data, &pos)
	lowDiff := getprice(data, &pos)
	// 服务端时间原始编码，交 `formatServerTime` 格式化。
	serverTimeRaw := getprice(data, &pos)
	// 特殊价格差值，语义待确认。
	negPriceRaw := getprice(data, &pos)
	// 总成交量与现量，均为变长整数解码。
	item.Vol = getprice(data, &pos)
	item.CurVol = getprice(data, &pos)
	// 成交额为 4 字节的单精度浮点数。
	item.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	// 内盘、外盘与两个统计字段，均为变长整数解码。
	item.InVol = getprice(data, &pos)
	item.OutVol = getprice(data, &pos)
	item.SAmount = getprice(data, &pos)
	item.OpenAmount = getprice(data, &pos)

	// 一档买卖：价格相对基准价偏移，档位量独立读取。
	bid := getprice(data, &pos) + basePrice
	ask := getprice(data, &pos) + basePrice
	bidVol := getprice(data, &pos)
	askVol := getprice(data, &pos)
	item.BidLevels = append(item.BidLevels, Level{Price: float64(bid) / 100.0, Vol: bidVol})
	item.AskLevels = append(item.AskLevels, Level{Price: float64(ask) / 100.0, Vol: askVol})

	// 尾部统计区：未知 2 字节字段，语义待确认。
	item.Unknown = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2
	// 涨速：的无符号整数转有符号短整数，放大 100 倍存储。
	riseSpeed := int16(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	// 短周期换手：同为放大 100 倍的有符号短整数。
	shortTurnover := int16(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	// 两分钟成交额，4 字节的单精度浮点数。
	min2Amount := math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	// 开盘抢筹指标：放大 100 倍的有符号短整数。
	openingRush := int16(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	// 10 字节协议保留区，整体跳过。
	pos += 10
	// 量涨速，4 字节的单精度浮点数。
	volRiseSpeed := math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	// 委比或深度指标，4 字节的单精度浮点数，语义待确认。
	depth := math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	// 24 字节协议保留区，整体跳过。
	pos += 24
	// 活跃度字段 2，2 字节`uint16`，语义待确认。
	item.Active2 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2

	// 基准价加差值还原各价位，除以 100 得到元。
	item.Close = float64(basePrice) / 100.0
	item.Price = item.Close
	item.PreClose = float64(basePrice+preCloseDiff) / 100.0
	item.Open = float64(basePrice+openDiff) / 100.0
	item.High = float64(basePrice+highDiff) / 100.0
	item.Low = float64(basePrice+lowDiff) / 100.0
	item.ServerTime = formatServerTime(serverTimeRaw)
	item.NegPrice = float64(negPriceRaw) / 100.0
	// 速度与换手类指标为放大 100 倍的有符号整数，解析后除以 100。
	item.RiseSpeed = float64(riseSpeed) / 100.0
	item.ShortTurnover = float64(shortTurnover) / 100.0
	item.Min2Amount = float64(min2Amount)
	item.OpeningRush = float64(openingRush) / 100.0
	item.VolRiseSpeed = float64(volRiseSpeed)
	item.Depth = float64(depth)

	return item, pos, nil
}
