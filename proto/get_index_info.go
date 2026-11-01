package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

// GetIndexInfo 封装指数实时概况（点位、涨跌、成交及上涨、下跌家数、档位分布）查询的请求与响应处理流程。
type GetIndexInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetIndexInfoRequest
	reply      *GetIndexInfoReply
}

// GetIndexInfoRequest 是指数实时概况查询的请求参数。
type GetIndexInfoRequest struct {
	Market uint16  // 市场代码（2 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长指数代码，尾部以零字节填充。
	Zero   uint32  // 协议保留占位，发送时填 0，语义待确认。
}

// GetIndexInfoReply 是指数实时概况查询的响应数据。
type GetIndexInfoReply struct {
	OrderCount uint32           // 委托档位明细条数（4 字节）。
	Market     uint8            // 回显市场代码（1 字节）。
	Code       string           // 回显指数代码。
	Active     uint16           // 活跃度，统计口径语义待确认。
	Close      float64          // 最新点位；点位按百分之一刻度存储，解析时已除以 100 还原。
	PreClose   float64          // 昨收点位，刻度口径同最新点位。
	Diff       float64          // 涨跌值（最新点位减昨收点位）。
	Open       float64          // 开盘点位，刻度口径同最新点位。
	High       float64          // 最高点位，刻度口径同最新点位。
	Low        float64          // 最低点位，刻度口径同最新点位。
	ServerTime string           // 服务端时间，由报文中的编码整数格式化得到。
	AfterHour  int              // 盘后字段或扩展标记，语义待确认。
	Vol        int              // 总成交量，单位口径以协议为准。
	CurVol     int              // 现量，即最近一笔成交量。
	Amount     float64          // 总成交额；协议内为 4 字节浮点位模式。
	OpenAmount int              // 开盘金额，口径语义待确认。
	UpCount    int              // 上涨家数。
	DownCount  int              // 下跌家数。
	Orders     []IndexInfoOrder // 解析后的档位委托分布明细。
}

// IndexInfoOrder 表示一条指数档位（委托分布）记录，含档位价格与挂单量。
type IndexInfoOrder struct {
	Price   float64 // 档位点位；协议内为累计差值，解析时累加后除以 100 还原。
	Unknown int     // 未确认字段，语义待确认。
	Vol     int     // 该档位挂单量，单位口径以协议为准。
}

// NewGetIndexInfo 创建并初始化指数实时概况查询请求对象。
//
// 参数 `req` 携带 `Market` 市场代码与 `Code` 指数代码；传 `nil` 时仅初始化协议骨架。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetIndexInfo(req *GetIndexInfoRequest) *GetIndexInfo {
	obj := new(GetIndexInfo)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetIndexInfoRequest)
	obj.reply = new(GetIndexInfoReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_INDEXINFO
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetIndexInfo) applyRequest(req *GetIndexInfoRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetIndexInfo) BuildRequest() ([]byte, error) {
	// 包头定长 0x0e 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 0x0e
	obj.reqHeader.PkgLen2 = 0x0e

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的指数实时概况与档位分布并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体；固定头字段读取失败时返回错误。
// 固定头为委托条数、市场、代码、活跃度；其后是按协议顺序排列的变长快照字段，
// 价格类字段存为相对最新点位的差值，快照读完后统一还原绝对点位。
func (obj *GetIndexInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	// 0:4 委托档位明细条数。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &obj.reply.OrderCount); err != nil {
		return err
	}
	pos += 4
	// 1 字节市场代码。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+1]), binary.LittleEndian, &obj.reply.Market); err != nil {
		return err
	}
	pos++
	// 6 字节定长指数代码。
	obj.reply.Code = Utf8ToGbk(data[pos : pos+6])
	pos += 6
	// 2 字节活跃度。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Active); err != nil {
		return err
	}
	pos += 2

	// 快照变长字段序列：最新点位、昨收差值、开盘差值、最高差值、最低差值、服务端时间、盘后标记。
	closeRaw := getprice(data, &pos)
	preCloseDiff := getprice(data, &pos)
	openDiff := getprice(data, &pos)
	highDiff := getprice(data, &pos)
	lowDiff := getprice(data, &pos)
	serverTimeRaw := getprice(data, &pos)
	afterHourRaw := getprice(data, &pos)
	// 总成交量、现量。
	obj.reply.Vol = getprice(data, &pos)
	obj.reply.CurVol = getprice(data, &pos)
	// 成交额为 4 字节浮点位模式，不经过变长编码。
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4

	// 剩余变长字段按顺序消费推进读指针；其中三项分别映射到开盘金额与上涨、下跌家数，其余不写入响应。
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	obj.reply.OpenAmount = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	obj.reply.UpCount = getprice(data, &pos)
	obj.reply.DownCount = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)
	_ = getprice(data, &pos)

	// 点位统一按百分之一刻度还原：最新点位即最新值，其余点位由最新点位叠加差值得到。
	obj.reply.Close = float64(closeRaw) / 100.0
	obj.reply.PreClose = float64(closeRaw+preCloseDiff) / 100.0
	// 涨跌值为最新点位减昨收；协议中昨收差值取反编码，故这里取负号还原。
	obj.reply.Diff = float64(-preCloseDiff) / 100.0
	obj.reply.Open = float64(closeRaw+openDiff) / 100.0
	obj.reply.High = float64(closeRaw+highDiff) / 100.0
	obj.reply.Low = float64(closeRaw+lowDiff) / 100.0
	obj.reply.ServerTime = formatServerTime(serverTimeRaw)
	obj.reply.AfterHour = afterHourRaw

	// 档位价格为累计差值，逐档累加后除以 100 还原为档位价格。
	lastPrice := 0
	for i := uint32(0); i < obj.reply.OrderCount; i++ {
		priceRaw := getprice(data, &pos)
		unknown := getprice(data, &pos)
		vol := getprice(data, &pos)
		lastPrice += priceRaw
		obj.reply.Orders = append(obj.reply.Orders, IndexInfoOrder{
			Price:   float64(lastPrice) / 100.0,
			Unknown: unknown,
			Vol:     vol,
		})
	}

	return nil
}

// Response 返回解析完成后的指数实时概况响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetIndexInfo) Response() *GetIndexInfoReply {
	return obj.reply
}
