package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

type GetIndexInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetIndexInfoRequest
	reply      *GetIndexInfoReply
}

type GetIndexInfoRequest struct {
	Market uint16  // 市场代码。
	Code   [6]byte // 指数代码。
	Zero   uint32  // 保留字段。
}

type GetIndexInfoReply struct {
	OrderCount uint32           // 委托明细条数。
	Market     uint8            // 市场代码。
	Code       string           // 指数代码。
	Active     uint16           // 活跃度。
	Close      float64          // 最新值。
	PreClose   float64          // 昨收值。
	Diff       float64          // 涨跌值。
	Open       float64          // 开盘值。
	High       float64          // 最高值。
	Low        float64          // 最低值。
	ServerTime string           // 服务端时间。
	AfterHour  int              // 盘后字段或扩展标记。
	Vol        int              // 总成交量。
	CurVol     int              // 现量。
	Amount     float64          // 总成交额。
	OpenAmount int              // 开盘金额。
	UpCount    int              // 上涨家数。
	DownCount  int              // 下跌家数。
	Orders     []IndexInfoOrder // 委托分布明细。
}

type IndexInfoOrder struct {
	Price   float64 // 档位价格。
	Unknown int     // 未确认字段。
	Vol     int     // 档位挂单量。
}

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

func (obj *GetIndexInfo) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0e
	obj.reqHeader.PkgLen2 = 0x0e

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetIndexInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &obj.reply.OrderCount); err != nil {
		return err
	}
	pos += 4
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+1]), binary.LittleEndian, &obj.reply.Market); err != nil {
		return err
	}
	pos++
	obj.reply.Code = Utf8ToGbk(data[pos : pos+6])
	pos += 6
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Active); err != nil {
		return err
	}
	pos += 2

	closeRaw := getprice(data, &pos)
	preCloseDiff := getprice(data, &pos)
	openDiff := getprice(data, &pos)
	highDiff := getprice(data, &pos)
	lowDiff := getprice(data, &pos)
	serverTimeRaw := getprice(data, &pos)
	afterHourRaw := getprice(data, &pos)
	obj.reply.Vol = getprice(data, &pos)
	obj.reply.CurVol = getprice(data, &pos)
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4

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

	obj.reply.Close = float64(closeRaw) / 100.0
	obj.reply.PreClose = float64(closeRaw+preCloseDiff) / 100.0
	obj.reply.Diff = float64(-preCloseDiff) / 100.0
	obj.reply.Open = float64(closeRaw+openDiff) / 100.0
	obj.reply.High = float64(closeRaw+highDiff) / 100.0
	obj.reply.Low = float64(closeRaw+lowDiff) / 100.0
	obj.reply.ServerTime = formatServerTime(serverTimeRaw)
	obj.reply.AfterHour = afterHourRaw

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

func (obj *GetIndexInfo) Response() *GetIndexInfoReply {
	return obj.reply
}
