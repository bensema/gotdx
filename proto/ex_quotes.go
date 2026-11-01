package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// ExStock 表示待查询行情的扩展市场品种标识，由市场类别与证券代码组成。
type ExStock struct {
	Category uint8  // 市场类别，取值由扩展市场分类约定。
	Code     string // 证券代码，发送时补齐为 23 字节定长 GBK 编码。
}

// ExQuoteItem 表示一条扩展市场行情快照，包含开高低收、成交量额、五档买卖盘及若干协议保留字段。
//
// 标注"语义待确认"的字段协议含义尚未逆向确认，仅按原始位宽解析保存；字段填充由共享解码器完成。
type ExQuoteItem struct {
	Category      uint8    // 市场类别。
	Code          string   // 证券代码，GBK 解码。
	Active        uint32   // 交易状态/活跃标志，`uint32`，语义待确认。
	PreClose      float64  // 昨收价，由 `float32` 还原，单位元。
	Open          float64  // 开盘价，由 `float32` 还原，单位元。
	High          float64  // 最高价，由 `float32` 还原，单位元。
	Low           float64  // 最低价，由 `float32` 还原，单位元。
	Close         float64  // 最新价/收盘价，由 `float32` 还原，单位元。
	OpenPosition  int      // 持仓量（期货类品种），由`uint32` 还原。
	AddPosition   int      // 持仓增减（期货类品种），由`uint32` 还原。
	Vol           int      // 累计成交量，由`uint32` 还原，计数单位由市场类型约定。
	CurVol        int      // 当前成交量，由`uint32` 还原。
	Amount        float64  // 成交额，由 `float32` 还原，单位元。
	InVol         int      // 内盘成交量，由`uint32` 还原。
	OutVol        int      // 外盘成交量，由`uint32` 还原。
	HoldPosition  int      // 持仓量，由`uint32` 还原，语义待确认。
	Unknown14     int      // 未知字段，由`uint32` 还原，语义待确认。
	BidLevels     []Level  // 五档买盘，价格与成交量各五档，按档位顺序排列。
	AskLevels     []Level  // 五档卖盘，价格与成交量各五档，按档位顺序排列。
	Settlement    float64  // 结算价（期货类品种），由 `float32` 还原。
	Avg           float64  // 均价，由 `float32` 还原，单位元。
	PreSettlement float64  // 前结算价，由 `float32` 还原。
	PreVol        float64  // 前成交量，由 `float32` 还原，语义待确认。
	Day3Raise     float64  // 三日涨幅，由 `float32` 还原，语义待确认。
	Settlement2   float64  // 第二结算价，由 `float32` 还原，语义待确认。
	Date          string   // 行情日期，格式为 `YYYY-MM-DD`；日期编号为 0 时取固定基准日期 1900-01-01。
	RaiseSpeed    float64  // 涨速，由 `float32` 还原。
	Unknown1      uint16   // 未知字段，`uint16`，语义待确认。
	Unknown2      uint32   // 未知字段，`uint32`，语义待确认。
	Unknown3      []uint32 // 未知字段组，连续四个`uint32`，语义待确认。
	Unknown7      float64  // 未知字段，由 `float32` 还原，语义待确认。
	Unknown8      float64  // 未知字段，由 `float32` 还原，语义待确认。
	Unknown9      uint32   // 未知字段，`uint32`，语义待确认。
	Unknown10     float64  // 未知字段，由 `float32` 还原，语义待确认。
	Unknown11     uint16   // 未知字段，`uint16`，语义待确认。
	Unknown12     uint8    // 未知字段，语义待确认。
}

// ExGetQuotesList 表示扩展市场行情列表分页查询协议，按分类与排序方式返回行情快照列表。
type ExGetQuotesList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuotesListRequest
	reply      *ExGetQuotesListReply
}

// ExGetQuotesListRequest 保存行情列表分页查询的品种分类、排序与起始偏移参数。
type ExGetQuotesListRequest struct {
	Category    uint8  // 品种分类，取值由扩展市场分类约定。
	SortType    uint16 // 排序字段编号，`uint16`，具体编号表语义待确认。
	Start       uint16 // 起始偏移条数，`uint16`，0 表示从第一条开始。
	Count       uint16 // 本页请求条数，`uint16`。
	SortReverse uint16 // 排序方向开关，`uint16`，取值与正倒序的对应关系语义待确认。
}

// ExGetQuotesListReply 保存解码完成的行情列表响应结果。
type ExGetQuotesListReply struct {
	Count uint16        // 返回条数，`uint16`。
	List  []ExQuoteItem // 解码出的行情快照列表。
}

// NewExGetQuotesList 构造行情列表查询协议实例。
//
// 参数 `req` 指定品种分类、排序字段、起始偏移、请求条数与排序方向；传入非空对象时覆盖默认参数，
// 传 `nil` 时使用空请求。返回的实例零值不可直接使用。
func NewExGetQuotesList(req *ExGetQuotesListRequest) *ExGetQuotesList {
	obj := &ExGetQuotesList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuotesListRequest),
		reply:      new(ExGetQuotesListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTESLIST
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetQuotesList) applyRequest(req *ExGetQuotesListRequest) {
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetQuotesList) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXQUOTESLIST, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码行情列表并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应头部不足 10 字节或快照条目
// 越界时返回非空错误，解码成功返回 `nil`。
func (obj *ExGetQuotesList) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	return parseExQuotesReply(data, &obj.reply.Count, &obj.reply.List)
}

// Response 返回解码完成的行情列表响应对象。
func (obj *ExGetQuotesList) Response() *ExGetQuotesListReply {
	return obj.reply
}

// ExGetQuote 表示单个扩展市场品种最新行情快照查询协议。
type ExGetQuote struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuoteRequest
	reply      *ExGetQuoteReply
}

// ExGetQuoteRequest 保存单品种行情查询的市场类别与证券代码参数。
type ExGetQuoteRequest struct {
	Category uint8   // 市场类别。
	Code     [9]byte // 证券代码，GBK 定长 9 字节，不足补零。
}

// ExGetQuoteReply 保存解码完成的单品种行情快照结果。
type ExGetQuoteReply struct {
	Item ExQuoteItem // 服务端返回的品种行情快照。
}

// NewExGetQuote 构造单品种行情查询协议实例。
//
// 参数 `req` 指定市场类别与证券代码；传入非空对象时覆盖默认参数，传 `nil` 时使用空请求。返回的
// 实例零值不可直接使用。
func NewExGetQuote(req *ExGetQuoteRequest) *ExGetQuote {
	obj := &ExGetQuote{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuoteRequest),
		reply:      new(ExGetQuoteReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTESINGLE
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetQuote) applyRequest(req *ExGetQuoteRequest) {
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetQuote) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXQUOTESINGLE, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码单品种行情快照并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；快照长度不足最小布局时返回非空
// 错误，解码成功返回 `nil`。
func (obj *ExGetQuote) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 单品种代码定长 9 字节，按对应布局解码。
	item, err := parseExQuoteItem(data, 9)
	if err != nil {
		return err
	}
	obj.reply.Item = item
	return nil
}

// Response 返回解码完成的单品种行情响应对象。
func (obj *ExGetQuote) Response() *ExGetQuoteReply {
	return obj.reply
}

// ExGetQuotes 表示批量扩展市场品种最新行情快照查询协议，一次请求多个品种。
type ExGetQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuotesRequest
	reply      *ExGetQuotesReply
}

// ExGetQuotesRequest 保存批量行情查询的待查询品种列表。
type ExGetQuotesRequest struct {
	Stocks []ExStock // 待查询品种标识列表；空列表时请求品种数为 0。
}

// ExGetQuotesReply 保存解码完成的批量行情响应结果。
type ExGetQuotesReply struct {
	Count uint16        // 返回条数，`uint16`。
	List  []ExQuoteItem // 解码出的行情快照列表。
}

// NewExGetQuotes 构造批量行情查询协议实例。
//
// 参数 `req` 指定待查询品种列表；传入非空对象时覆盖默认参数，传 `nil` 时使用空请求。返回的实例
// 零值不可直接使用。
func NewExGetQuotes(req *ExGetQuotesRequest) *ExGetQuotes {
	obj := &ExGetQuotes{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuotesRequest),
		reply:      new(ExGetQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetQuotes) applyRequest(req *ExGetQuotesRequest) {
	obj.request = req
}

// BuildRequest 把待查询品种列表编码为批量行情请求字节流。
//
// 返回值第一项为编码后的请求字节流；前缀段或品种条目序列化失败时第二项返回非空错误。
func (obj *ExGetQuotes) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 固定前缀：类别标识 5。
	if err := binary.Write(payload, binary.LittleEndian, uint8(5)); err != nil {
		return nil, err
	}
	// 7 字节协议保留零填充。
	if _, err := payload.Write(make([]byte, 7)); err != nil {
		return nil, err
	}
	// 品种数量，`uint16`。
	if err := binary.Write(payload, binary.LittleEndian, uint16(len(obj.request.Stocks))); err != nil {
		return nil, err
	}
	for _, stock := range obj.request.Stocks {
		// 每个品种先写市场类别。
		if err := binary.Write(payload, binary.LittleEndian, stock.Category); err != nil {
			return nil, err
		}
		// 代码补齐为 23 字节定长，不足部分补零，超出部分截断。
		code := make([]byte, 23)
		copy(code, stock.Code)
		if _, err := payload.Write(code); err != nil {
			return nil, err
		}
	}
	return buildExRequest(KMSG_EXQUOTES, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码批量行情快照并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应头部不足 10 字节或快照条目
// 越界时返回非空错误，解码成功返回 `nil`。
func (obj *ExGetQuotes) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	return parseExQuotesReply(data, &obj.reply.Count, &obj.reply.List)
}

// Response 返回解码完成的批量行情响应对象。
func (obj *ExGetQuotes) Response() *ExGetQuotesReply {
	return obj.reply
}

// ExGetQuotes2 表示批量行情查询的第二种协议变体，请求与响应结构同 ExGetQuotes，仅请求前缀报文格式不同。
type ExGetQuotes2 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuotesRequest
	reply      *ExGetQuotesReply
}

// NewExGetQuotes2 构造第二种批量行情协议实例。
//
// 参数 `req` 指定待查询品种列表；传入非空对象时覆盖默认参数，传 `nil` 时使用空请求。返回的实例
// 零值不可直接使用。
func NewExGetQuotes2(req *ExGetQuotesRequest) *ExGetQuotes2 {
	obj := &ExGetQuotes2{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuotesRequest),
		reply:      new(ExGetQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES2
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetQuotes2) applyRequest(req *ExGetQuotesRequest) {
	obj.request = req
}

// BuildRequest 按第二种批量行情协议格式编码请求字节流。
//
// 返回值第一项为编码后的请求字节流；前缀段或品种条目序列化失败时第二项返回非空错误。
func (obj *ExGetQuotes2) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 固定前缀四个`uint16` 参数（协议约定值，含义待确认）。
	if err := binary.Write(payload, binary.LittleEndian, uint16(2)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(3148)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(600)); err != nil {
		return nil, err
	}
	// 品种数量，`uint16`。
	if err := binary.Write(payload, binary.LittleEndian, uint16(len(obj.request.Stocks))); err != nil {
		return nil, err
	}
	for _, stock := range obj.request.Stocks {
		// 每个品种先写市场类别。
		if err := binary.Write(payload, binary.LittleEndian, stock.Category); err != nil {
			return nil, err
		}
		// 代码补齐为 23 字节定长，不足补零，超出截断。
		code := make([]byte, 23)
		copy(code, stock.Code)
		if _, err := payload.Write(code); err != nil {
			return nil, err
		}
	}
	return buildExRequest(KMSG_EXQUOTES2, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码第二种协议的批量行情快照并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应头部不足 10 字节或快照条目
// 越界时返回非空错误，解码成功返回 `nil`。
func (obj *ExGetQuotes2) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	return parseExQuotesReply(data, &obj.reply.Count, &obj.reply.List)
}

// Response 返回第二种协议解码完成的批量行情响应对象。
func (obj *ExGetQuotes2) Response() *ExGetQuotesReply {
	return obj.reply
}

// parseExQuotesReply 是行情列表与批量快照的共享响应解码器，按定长 314 字节逐条解码。
//
// 参数 `data` 为服务端返回的原始响应字节流；`count` 为输出参数，回填响应头部给出的返回条数；`out`
// 为输出参数，追加解码出的行情快照。响应头部不足 10 字节或任一条目越界时返回非空错误，解码
// 成功返回 `nil`。
func parseExQuotesReply(data []byte, count *uint16, out *[]ExQuoteItem) error {
	// 头部至少需要 10 字节：前 8 字节为保留区，8:10 为`uint16` 条数。
	if len(data) < 10 {
		return fmt.Errorf("invalid ex quotes response length: %d", len(data))
	}
	*count = binary.LittleEndian.Uint16(data[8:10])
	for i := uint16(0); i < *count; i++ {
		// 每条快照定长 314 字节，先校验剩余长度防止越界。
		base := 10 + int(i)*314
		if base+314 > len(data) {
			return fmt.Errorf("invalid ex quotes item %d", i)
		}
		// 列表与批量变体代码定长 23 字节，按对应布局解码。
		item, err := parseExQuoteItem(data[base:base+314], 23)
		if err != nil {
			return err
		}
		*out = append(*out, item)
	}
	return nil
}

// parseExQuoteItem 解码单条行情快照。
//
// 参数 `data` 为单条快照的原始字节；`codeLen` 指定代码字段定长（单品种为 9，列表与批量为 23）。
// 返回值第一项为解码出的快照；快照长度不足 `291+codeLen` 最小布局时第二项返回非空错误。
func parseExQuoteItem(data []byte, codeLen int) (ExQuoteItem, error) {
	// 最小长度为代码之前固定区 291 字节加上代码长度，先做整体边界防护。
	minLen := 291 + codeLen
	if len(data) < minLen {
		return ExQuoteItem{}, fmt.Errorf("invalid ex quote item length: %d", len(data))
	}

	item := ExQuoteItem{
		// 首字节为市场类别，随后 `codeLen` 字节为 GBK 证券代码。
		Category: data[0],
		Code:     Utf8ToGbk(data[1 : 1+codeLen]),
	}
	// 跳过类别与代码，定位到后续数值字段区。
	pos := 1 + codeLen

	// 以下 15 个字段均为 4 字节：`uint32` 状态/持仓/成交量类，`float32` 价格/金额类。
	item.Active = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.OpenPosition = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.AddPosition = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.Vol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.CurVol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.InVol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.OutVol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.Unknown14 = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.HoldPosition = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4

	// 五档买盘：先批量读出 5 个价格（每档 4 字节，共 20 字节）。
	for i := 0; i < 5; i++ {
		price := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4])))
		item.BidLevels = append(item.BidLevels, Level{Price: price})
	}
	pos += 20
	// 再批量回填 5 个买盘成交量（每档 4 字节，共 20 字节）。
	for i := 0; i < 5; i++ {
		item.BidLevels[i].Vol = int(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4]))
	}
	pos += 20
	// 五档卖盘价格段，布局同买盘。
	for i := 0; i < 5; i++ {
		price := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4])))
		item.AskLevels = append(item.AskLevels, Level{Price: price})
	}
	pos += 20
	// 五档卖盘成交量段。
	for i := 0; i < 5; i++ {
		item.AskLevels[i].Vol = int(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4]))
	}
	pos += 20

	item.Unknown1 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2
	item.Settlement = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Unknown2 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.Avg = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.PreSettlement = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	// 连续 4 个 `u32` 保留/未知字段，整体读入 `Unknown3`。
	item.Unknown3 = append(item.Unknown3,
		binary.LittleEndian.Uint32(data[pos:pos+4]),
		binary.LittleEndian.Uint32(data[pos+4:pos+8]),
		binary.LittleEndian.Uint32(data[pos+8:pos+12]),
		binary.LittleEndian.Uint32(data[pos+12:pos+16]),
	)
	pos += 16
	item.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4

	// 跳过一段协议填充/未知区。
	pos += 12
	item.PreVol = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Unknown7 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	// 跳过下一段协议填充/未知区。
	pos += 12
	item.Unknown8 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Day3Raise = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	// 跳过协议保留填充区。
	pos += 25
	item.Settlement2 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	dateRaw := binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.Unknown9 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.RaiseSpeed = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Unknown10 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	// 跳过协议保留填充区。
	pos += 24
	item.Unknown11 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2
	item.Unknown12 = data[pos]

	// 日期编号为 0 表示无有效日期，使用固定基准日期 1900-01-01；
	// 否则按 `YYYYMMDD` 拆分成本地时间并格式化为日期文本。
	if dateRaw == 0 {
		item.Date = "1900-01-01"
	} else {
		d := time.Date(int(dateRaw/10000), time.Month((dateRaw%10000)/100), int(dateRaw%100), 0, 0, 0, 0, time.Local)
		item.Date = d.Format("2006-01-02")
	}

	return item, nil
}
