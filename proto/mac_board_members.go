package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// MACBoardMembers 查询指定板块的成分股列表，仅返回市场、代码与名称。
//
// 使用时先由 NewMACBoardMembers 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACBoardMembers struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardMembersRequest
	reply      *MACBoardMembersReply
}

// MACBoardMembersRequest 保存成分股列表查询的板块代码、排序、起始位置与分页参数。
type MACBoardMembersRequest struct {
	BoardCode uint32   // 板块代码，由板块符号经协议映射得到，四字节。
	Reserved1 [9]byte  // 协议保留填充字节，填零。
	SortType  uint16   // 排序字段编号，取值见根包 `MACSortType*` 常量；为 0 时默认回填 14（`MACSortChangePct`，按涨跌幅排序）。
	Start     uint32   // 起始成分股序号（分页偏移），四字节。
	PageSize  uint8    // 每页条数，为 0 时默认取 80。
	Zero      uint8    // 协议保留字节，语义待确认。
	SortOrder uint16   // 排序方向编号，取值见根包 `MACSortOrder*` 常量（1 为降序、2 为升序、0 为不指定），为 0 时默认回填 1。
	Extra     [20]byte // 扩展请求段（排序/过滤参数），具体位域语义待确认。
}

// MACBoardMembersReply 保存成分股列表查询返回的板块名称、计数与逐条成分股。
type MACBoardMembersReply struct {
	Name   string               // 板块名称，GBK 解码。
	Count  uint16               // 本次返回的成分股条数，两字节。
	Total  uint32               // 板块成分股总数，四字节。
	Stocks []MACBoardMemberItem // 成分股列表，按排序结果顺序排列。
}

// MACBoardMemberItem 表示一只板块成分股的市场、代码与名称。
type MACBoardMemberItem struct {
	Name   string // 成分股名称，GBK 解码。
	Market uint16 // 市场代码，两字节。
	Symbol string // 成分股代码，GBK 解码。
}

// NewMACBoardMembers 创建成分股列表查询对象；请求参数为空时使用内置默认参数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`BoardCode` 为板块代码、`SortType` 为排序字段、
// `Start` 为起始序号、`PageSize` 为每页条数、`SortOrder` 为排序方向；
// `SortType`、`PageSize`、`SortOrder` 为 0 时分别回填默认值 14、80、1。
func NewMACBoardMembers(req *MACBoardMembersRequest) *MACBoardMembers {
	obj := &MACBoardMembers{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardMembersRequest),
		reply:      new(MACBoardMembersReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACBOARDMEMBERS
	obj.request.SortType = 14
	obj.request.PageSize = 80
	obj.request.SortOrder = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACBoardMembers) applyRequest(req *MACBoardMembersRequest) {
	if req.PageSize == 0 {
		req.PageSize = 80
	}
	if req.SortType == 0 {
		req.SortType = 14
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACBOARDMEMBERS` 与请求头封装成完整报文。
func (obj *MACBoardMembers) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACBOARDMEMBERS, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACBoardMembers) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 响应头不足 26 字节时直接返回错误，避免后续切片越界。
	if len(data) < 26 {
		return fmt.Errorf("invalid mac board members response length: %d", len(data))
	}

	// 头部布局：GBK 板块名称（16:20）、成分股总数（20:24）、本次条数（24:26），均读取。
	obj.reply.Name = Utf8ToGbk(data[16:20])
	obj.reply.Total = binary.LittleEndian.Uint32(data[20:24])
	obj.reply.Count = binary.LittleEndian.Uint16(data[24:26])

	// 记录区自偏移 26 开始，每条定长 68 字节。
	pos := 26
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 剩余字节不足一条记录时中止并报错，防止越界读取。
		if pos+68 > len(data) {
			return fmt.Errorf("invalid mac board member item %d", i)
		}
		// 记录布局：市场（0:2）、GBK 代码（2:8）、GBK 名称（24:40），中间偏移为填充或保留区。
		item := MACBoardMemberItem{
			Market: binary.LittleEndian.Uint16(data[pos : pos+2]),
			Symbol: Utf8ToGbk(data[pos+2 : pos+8]),
			Name:   Utf8ToGbk(data[pos+24 : pos+40]),
		}
		obj.reply.Stocks = append(obj.reply.Stocks, item)
		pos += 68
	}

	return nil
}

// Response 返回解码完成的成分股列表响应；未调用 ParseResponse 前内容为零值。
func (obj *MACBoardMembers) Response() *MACBoardMembersReply {
	return obj.reply
}

// MACBoardMembersQuotes 查询指定板块成分股的固定字段实时报价快照。
//
// 使用时先由 NewMACBoardMembersQuotes 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACBoardMembersQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardMembersQuotesRequest
	reply      *MACBoardMembersQuotesReply
}

// MACBoardMembersQuotesRequest 保存成分股报价查询的板块代码、排序、分页与字段位图参数。
type MACBoardMembersQuotesRequest struct {
	BoardCode uint32   // 板块代码，由板块符号经协议映射得到，四字节。
	Reserved1 [9]byte  // 协议保留填充字节，填零。
	SortType  uint16   // 排序字段编号，取值见根包 `MACSortType*` 常量；为 0 时默认回填 14（`MACSortChangePct`，按涨跌幅排序）。
	Start     uint32   // 起始成分股序号（分页偏移），四字节。
	PageSize  uint8    // 每页条数，为 0 时默认取 80。
	Zero      uint8    // 协议保留字节，语义待确认。
	SortOrder uint8    // 排序方向编号，取值见根包 `MACSortOrder*` 常量（1 为降序、2 为升序、0 为不指定），为 0 时默认回填 1。
	Extra     [21]byte // 固定字段位图与过滤参数段，使用内置默认字节序列，逐位语义待确认。
}

// MACBoardMembersQuotesReply 保存成分股报价查询返回的板块名称、计数与逐条报价。
type MACBoardMembersQuotesReply struct {
	Name   string                    // 板块名称，GBK 解码。
	Count  uint16                    // 本次返回的成分股条数，两字节。
	Total  uint32                    // 板块成分股总数，四字节。
	Stocks []MACBoardMemberQuoteItem // 成分股报价列表，按排序结果顺序排列。
}

// MACBoardMemberQuoteItem 表示一只成分股的固定字段实时报价快照。
//
// 槽位顺序与服务端 128 字节指标区一一对应；部分占位字段与别名字段的原始语义待确认，
// 实现未据此推断额外协议含义。
type MACBoardMemberQuoteItem struct {
	Name               string  // 成分股名称，GBK 解码。
	Market             uint16  // 市场代码，两字节。
	Symbol             string  // 成分股代码，GBK 解码。
	PreClose           float64 // 昨收价，单精度浮点还原。
	Open               float64 // 开盘价，单精度浮点还原。
	High               float64 // 最高价，单精度浮点还原。
	Low                float64 // 最低价，单精度浮点还原。
	Close              float64 // 收盘价，单精度浮点还原。
	Unknown6           float64 // 槽位 6 的浮点副本，语义待确认。
	Vol                uint32  // 成交量，四字节无符号整数。
	VolumeRatio        float64 // 量比，单精度浮点还原。
	Amount             float64 // 成交额，单精度浮点还原。
	TotalShares        float64 // 总股本，单精度浮点还原。
	FloatShares        float64 // 流通股本，单精度浮点还原。
	EPS                float64 // 每股收益，单精度浮点还原。
	ROE                float64 // 净资产收益率，复用每股净资产槽位，口径语义待确认。
	NetAssets          float64 // 每股净资产，单精度浮点还原。
	ActionPrice        float64 // 槽位 12 指标，原始语义待确认。
	Unknown13          float64 // 槽位 12 别名字段，语义待确认。
	UnknownActionPrice float64 // 槽位 12 别名字段，语义待确认。
	MarketCap          float64 // 总市值，复用总市值槽位。
	TotalMarketCapAB   float64 // 总市值（含 A/B 股口径），单精度浮点还原。
	PEDynamic          float64 // 动态市盈率，单精度浮点还原。
	Zero16             float64 // 槽位 15 的浮点副本，语义待确认。
	LotSizeInfo        uint32  // 槽位 15 整数值，具体含义待确认。
	Unknown23          float64 // 槽位 16 指标，语义待确认。
	Zero17             float64 // 槽位 16 别名字段，语义待确认。
	DividendYield      float64 // 股息率，单精度浮点还原。
	RiseSpeed          float64 // 涨速，复用涨速槽位。
	CurrentVol         uint16  // 现量，由现量四字节取低两字节得到。
	LastVolume         uint32  // 现量，四字节无符号整数。
	Turnover           float64 // 换手率，单精度浮点还原。
	TurnoverRate       float64 // 换手率，复用换手率槽位。
	Unknown21          float64 // 标签位图的浮点副本，语义待确认。
	SomeBitmap         uint32  // 标签位图，四字节，位定义语义待确认。
	Unknown22          float64 // 小数位标志的浮点副本，语义待确认。
	DecimalPoint       uint32  // 价格小数位数，四字节。
	LimitUp            float64 // 涨停价，复用买入限价槽位。
	BuyPriceLimit      float64 // 买入限价（涨停价），单精度浮点还原。
	LimitDown          float64 // 跌停价，复用卖出限价槽位。
	SellPriceLimit     float64 // 卖出限价（跌停价），单精度浮点还原。
	Zero25             float64 // 槽位 24 的浮点副本，语义待确认。
	Unknown34          uint32  // 槽位 24 整数值，语义待确认。
	Unknown26          float64 // 每手股数的浮点副本，语义待确认。
	LotSize            uint32  // 每手股数，四字节。
	LotSizeBoardSymbol string  // 由每手股数换算得到的板块标识代码。
	Unknown27          float64 // 槽位 26 别名字段，语义待确认。
	PreIPOV            float64 // 槽位 26 指标，原始语义待确认。
	RiseSpeed2         float64 // 涨速，复用涨速槽位的第二别名。
	SpeedPct           float64 // 涨速百分比，单精度浮点还原。
	Zero29             float64 // 科创板标志的浮点副本，语义待确认。
	FlagKCB            uint32  // 科创板标志，复用科创板标志槽位。
	KCBFlag            uint32  // 科创板标志，四字节无符号整数。
	PEStatic           float64 // 静态市盈率，单精度浮点还原。
	PETTM              float64 // 滚动市盈率（TTM），单精度浮点还原。
	Unknown31          float64 // 末槽位别名字段，语义待确认。
	UnknownClosePrice  float64 // 末槽位指标，原始语义待确认。
}

// NewMACBoardMembersQuotes 创建成分股报价查询对象；请求参数为空时使用内置默认参数与默认位图。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`BoardCode` 为板块代码、`SortType` 为排序字段、
// `Start` 为起始序号、`PageSize` 为每页条数、`SortOrder` 为排序方向；
// `SortType`、`PageSize`、`SortOrder` 为 0 时分别回填默认值 14、80、1，`Extra` 为空时使用内置默认位图。
func NewMACBoardMembersQuotes(req *MACBoardMembersQuotesRequest) *MACBoardMembersQuotes {
	obj := &MACBoardMembersQuotes{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardMembersQuotesRequest),
		reply:      new(MACBoardMembersQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACBOARDMEMBERS
	obj.request.SortType = 14
	obj.request.PageSize = 80
	obj.request.SortOrder = 1
	obj.request.Extra = [21]byte{
		0x00, 0xff, 0xfc, 0xe1, 0xcc, 0x3f, 0x08, 0x03, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00,
	}
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACBoardMembersQuotes) applyRequest(req *MACBoardMembersQuotesRequest) {
	if req.PageSize == 0 {
		req.PageSize = 80
	}
	if req.SortType == 0 {
		req.SortType = 14
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	if req.Extra == ([21]byte{}) {
		req.Extra = obj.request.Extra
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；逐字段写入失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数。由于结构体内存对齐与线上字节布局不一致，
// 不能整体序列化，需按线上字段顺序逐字段手工写入，最后由 `buildExRequest` 追加命令码
// `KMSG_MACBOARDMEMBERS` 与请求头封装成完整报文。
func (obj *MACBoardMembersQuotes) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 结构体对齐与线上字节布局不一致，不能整体序列化，需按线上字段顺序逐字段手工写入。
	if err := binary.Write(payload, binary.LittleEndian, obj.request.BoardCode); err != nil {
		return nil, err
	}
	// 9 字节保留区原样透传。
	if _, err := payload.Write(obj.request.Reserved1[:]); err != nil {
		return nil, err
	}
	// 排序类型、起始偏移、分页大小、保留字节、排序方向依次按线上顺序写入。
	if err := binary.Write(payload, binary.LittleEndian, obj.request.SortType); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.Start); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.PageSize); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.Zero); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.SortOrder); err != nil {
		return nil, err
	}
	// 21 字节位图扩展段整体写入。
	if _, err := payload.Write(obj.request.Extra[:]); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_MACBOARDMEMBERS, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACBoardMembersQuotes) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 响应头不足 26 字节时直接返回错误，避免后续切片越界。
	if len(data) < 26 {
		return fmt.Errorf("invalid mac board members quotes response length: %d", len(data))
	}

	// 头部布局：GBK 板块名称（16:20）、成分股总数（20:24）、本次条数（24:26）。
	obj.reply.Name = Utf8ToGbk(data[16:20])
	obj.reply.Total = binary.LittleEndian.Uint32(data[20:24])
	obj.reply.Count = binary.LittleEndian.Uint16(data[24:26])

	pos := 26
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 剩余字节不足一条记录时中止并报错，防止越界读取。
		if pos+196 > len(data) {
			return fmt.Errorf("invalid mac board member quote item %d", i)
		}
		// 身份段：市场（0:2）、GBK 代码（2:8）、GBK 名称（24:48）。
		item := MACBoardMemberQuoteItem{
			Market: binary.LittleEndian.Uint16(data[pos : pos+2]),
			Symbol: Utf8ToGbk(data[pos+2 : pos+8]),
			Name:   Utf8ToGbk(data[pos+24 : pos+48]),
		}

		// 指标区自偏移 68 起共 128 字节，每槽 4 字节，按槽位索引分别解码为浮点或计数。
		metrics := data[pos+68 : pos+196]
		// 闭包：将指定槽位的 4 字节还原为单精度浮点后再转双精度。
		floatAt := func(index int) float64 {
			return float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[index*4 : index*4+4])))
		}
		// 闭包：将指定槽位读取为四字节无符号整数，用于计数、位图与精度标志。
		uintAt := func(index int) uint32 {
			return binary.LittleEndian.Uint32(metrics[index*4 : index*4+4])
		}

		// 槽位 0 到 4 为昨收、开、高、低、收；槽位 5 为成交量。
		item.PreClose = floatAt(0)
		item.Open = floatAt(1)
		item.High = floatAt(2)
		item.Low = floatAt(3)
		item.Close = floatAt(4)
		item.Vol = uintAt(5)
		item.Unknown6 = float64(item.Vol)
		// 槽位 6 到 17 为量比、成交额、股本、净资产、市值、动态市盈率、股息率等基本面指标。
		item.VolumeRatio = floatAt(6)
		item.Amount = floatAt(7)
		item.TotalShares = floatAt(8)
		item.FloatShares = floatAt(9)
		item.EPS = floatAt(10)
		item.NetAssets = floatAt(11)
		item.ROE = item.NetAssets
		item.ActionPrice = floatAt(12)
		item.UnknownActionPrice = item.ActionPrice
		item.Unknown13 = item.ActionPrice
		item.TotalMarketCapAB = floatAt(13)
		item.MarketCap = item.TotalMarketCapAB
		item.PEDynamic = floatAt(14)
		item.LotSizeInfo = uintAt(15)
		item.Zero16 = float64(item.LotSizeInfo)
		item.Unknown23 = floatAt(16)
		item.Zero17 = item.Unknown23
		item.DividendYield = floatAt(17)
		// 槽位 27 为涨速；槽位 18 起为现量、换手、标签位图、价格精度、涨跌停价等盘口字段。
		// 该分组沿用服务端固定槽位顺序，个别槽位的精确语义仍待确认。
		item.SpeedPct = floatAt(27)
		item.RiseSpeed = item.SpeedPct
		item.LastVolume = uintAt(18)
		item.CurrentVol = uint16(item.LastVolume)
		item.Turnover = floatAt(19)
		item.TurnoverRate = item.Turnover
		item.SomeBitmap = uintAt(20)
		item.Unknown21 = float64(item.SomeBitmap)
		item.DecimalPoint = uintAt(21)
		item.Unknown22 = float64(item.DecimalPoint)
		item.BuyPriceLimit = floatAt(22)
		item.LimitUp = item.BuyPriceLimit
		item.SellPriceLimit = floatAt(23)
		item.LimitDown = item.SellPriceLimit
		item.Unknown34 = uintAt(24)
		item.Zero25 = float64(item.Unknown34)
		item.LotSize = uintAt(25)
		item.LotSizeBoardSymbol = macLotSizeBoardSymbol(item.LotSize)
		item.Unknown26 = float64(item.LotSize)
		item.PreIPOV = floatAt(26)
		item.Unknown27 = item.PreIPOV
		item.RiseSpeed2 = item.SpeedPct
		// 槽位 28 到 31 为板块标志、滚动市盈率、静态市盈率及末槽位，末槽位语义待确认。
		item.KCBFlag = uintAt(28)
		item.FlagKCB = item.KCBFlag
		item.Zero29 = float64(item.KCBFlag)
		item.PETTM = floatAt(29)
		item.PEStatic = floatAt(30)
		item.UnknownClosePrice = floatAt(31)
		item.Unknown31 = item.UnknownClosePrice

		obj.reply.Stocks = append(obj.reply.Stocks, item)
		pos += 196
	}

	return nil
}

// Response 返回解码完成的成分股报价响应；未调用 ParseResponse 前内容为零值。
func (obj *MACBoardMembersQuotes) Response() *MACBoardMembersQuotesReply {
	return obj.reply
}
