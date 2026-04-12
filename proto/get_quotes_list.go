package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

type GetQuotesList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetQuotesListRequest
	reply      *GetQuotesListReply
}

type GetQuotesListRequest struct {
	Category    uint16 // 板块或市场分类。
	SortType    uint16 // 排序方式。
	Start       uint16 // 起始偏移。
	Count       uint16 // 请求条数。
	SortReverse uint16 // 是否倒序排序。
	Mode        uint16 // 协议模式位。
	Filter      uint16 // 过滤条件。
	One         uint16 // 固定位，通常为 1。
	Zero        uint16 // 保留字段。
}

type GetQuotesListReply struct {
	Block uint16          // 区块或分组标识。
	Count uint16          // 返回条数。
	List  []QuoteListItem // 排序行情列表。
}

type QuoteListItem struct {
	Market        uint8   // 市场代码。
	Code          string  // 证券代码。
	Active1       uint16  // 活跃度字段 1。
	Active2       uint16  // 活跃度字段 2。
	Close         float64 // 最新价。
	Price         float64 // 当前价，通常与 Close 一致。
	PreClose      float64 // 昨收价。
	Open          float64 // 今开价。
	High          float64 // 最高价。
	Low           float64 // 最低价。
	ServerTime    string  // 服务端时间。
	NegPrice      float64 // 特殊价格字段。
	Vol           int     // 总成交量。
	CurVol        int     // 现量。
	Amount        float64 // 总成交额。
	InVol         int     // 内盘量。
	OutVol        int     // 外盘量。
	SAmount       int     // 上涨家数或卖出相关统计字段。
	OpenAmount    int     // 开盘金额。
	BidLevels     []Level // 买盘档位列表。
	AskLevels     []Level // 卖盘档位列表。
	Unknown       uint16  // 未确认字段。
	RiseSpeed     float64 // 涨速。
	ShortTurnover float64 // 短换手。
	Min2Amount    float64 // 两分钟成交额。
	OpeningRush   float64 // 开盘抢筹。
	VolRiseSpeed  float64 // 量涨速。
	Depth         float64 // 委比或深度指标。
	Turnover      float64 // 换手率，按高层接口 best-effort 补齐。
}

func NewGetQuotesList() *GetQuotesList {
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
	return obj
}

func (obj *GetQuotesList) SetParams(req *GetQuotesListRequest) {
	if req.Mode == 0 {
		req.Mode = 5
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *GetQuotesList) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 20
	obj.reqHeader.PkgLen2 = 20

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetQuotesList) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

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

func (obj *GetQuotesList) Reply() *GetQuotesListReply {
	return obj.reply
}

type GetQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetQuotesRequest
	reply      *GetQuotesReply
}

type GetQuotesRequest struct {
	Stocks []Stock // 待查询的证券列表。
}

type GetQuotesReply struct {
	Block uint16          // 区块或分组标识。
	Count uint16          // 返回条数。
	List  []QuoteListItem // 批量行情列表。
}

func NewGetQuotes() *GetQuotes {
	obj := new(GetQuotes)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetQuotesRequest)
	obj.reply = new(GetQuotesReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_QUOTES
	return obj
}

func (obj *GetQuotes) SetParams(req *GetQuotesRequest) {
	obj.request = req
}

func (obj *GetQuotes) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2 + 10 + uint16(len(obj.request.Stocks)*7)
	obj.reqHeader.PkgLen2 = 2 + 10 + uint16(len(obj.request.Stocks)*7)

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(5)); err != nil {
		return nil, err
	}
	if _, err := buf.Write(make([]byte, 6)); err != nil {
		return nil, err
	}
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

func (obj *GetQuotes) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

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

func (obj *GetQuotes) Reply() *GetQuotesReply {
	return obj.reply
}

func parseQuoteListItem(data []byte, pos int) (QuoteListItem, int, error) {
	item := QuoteListItem{}

	item.Market = data[pos]
	pos++
	item.Code = Utf8ToGbk(data[pos : pos+6])
	pos += 6
	item.Active1 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2

	basePrice := getprice(data, &pos)
	preCloseDiff := getprice(data, &pos)
	openDiff := getprice(data, &pos)
	highDiff := getprice(data, &pos)
	lowDiff := getprice(data, &pos)
	serverTimeRaw := getprice(data, &pos)
	negPriceRaw := getprice(data, &pos)
	item.Vol = getprice(data, &pos)
	item.CurVol = getprice(data, &pos)
	item.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.InVol = getprice(data, &pos)
	item.OutVol = getprice(data, &pos)
	item.SAmount = getprice(data, &pos)
	item.OpenAmount = getprice(data, &pos)

	bid := getprice(data, &pos) + basePrice
	ask := getprice(data, &pos) + basePrice
	bidVol := getprice(data, &pos)
	askVol := getprice(data, &pos)
	item.BidLevels = append(item.BidLevels, Level{Price: float64(bid) / 100.0, Vol: bidVol})
	item.AskLevels = append(item.AskLevels, Level{Price: float64(ask) / 100.0, Vol: askVol})

	item.Unknown = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2
	riseSpeed := int16(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	shortTurnover := int16(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	min2Amount := math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	openingRush := int16(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	pos += 10
	volRiseSpeed := math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	depth := math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	pos += 24
	item.Active2 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2

	item.Close = float64(basePrice) / 100.0
	item.Price = item.Close
	item.PreClose = float64(basePrice+preCloseDiff) / 100.0
	item.Open = float64(basePrice+openDiff) / 100.0
	item.High = float64(basePrice+highDiff) / 100.0
	item.Low = float64(basePrice+lowDiff) / 100.0
	item.ServerTime = formatServerTime(serverTimeRaw)
	item.NegPrice = float64(negPriceRaw) / 100.0
	item.RiseSpeed = float64(riseSpeed) / 100.0
	item.ShortTurnover = float64(shortTurnover) / 100.0
	item.Min2Amount = float64(min2Amount)
	item.OpeningRush = float64(openingRush) / 100.0
	item.VolRiseSpeed = float64(volRiseSpeed)
	item.Depth = float64(depth)

	return item, pos, nil
}
