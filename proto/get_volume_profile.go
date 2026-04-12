package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

type GetVolumeProfile struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetVolumeProfileRequest
	reply      *GetVolumeProfileReply
}

type GetVolumeProfileRequest struct {
	Market uint16  // 市场代码。
	Code   [6]byte // 证券代码。
}

type GetVolumeProfileReply struct {
	Count       uint16              // 成交分布档位数。
	Market      uint8               // 市场代码。
	Code        string              // 证券代码。
	Active      uint16              // 活跃度。
	Close       float64             // 最新价。
	Open        float64             // 今开价。
	High        float64             // 最高价。
	Low         float64             // 最低价。
	PreClose    float64             // 昨收价。
	ServerTime  string              // 服务端时间。
	NegPrice    float64             // 特殊价格字段。
	Vol         int                 // 总成交量。
	CurVol      int                 // 现量。
	Amount      float64             // 总成交额。
	InVol       int                 // 内盘量。
	OutVol      int                 // 外盘量。
	SAmount     int                 // 上涨家数或卖出相关统计字段。
	OpenAmount  int                 // 开盘金额。
	Turnover    float64             // 换手率，按高层接口 best-effort 补齐。
	BidLevels   []Level             // 买盘三档。
	AskLevels   []Level             // 卖盘三档。
	Unknown     uint16              // 未确认字段。
	VolProfiles []VolumeProfileItem // 成交分布明细。
}

type VolumeProfileItem struct {
	Price float64 // 档位价格。
	Vol   int     // 该档总成交量。
	Buy   int     // 主买量。
	Sell  int     // 主卖量。
}

func NewGetVolumeProfile() *GetVolumeProfile {
	obj := new(GetVolumeProfile)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetVolumeProfileRequest)
	obj.reply = new(GetVolumeProfileReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_VOLUMEPROFILE
	return obj
}

func (obj *GetVolumeProfile) SetParams(req *GetVolumeProfileRequest) {
	obj.request = req
}

func (obj *GetVolumeProfile) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0a
	obj.reqHeader.PkgLen2 = 0x0a

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetVolumeProfile) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Market = data[2]
	obj.reply.Code = Utf8ToGbk(data[3:9])
	obj.reply.Active = binary.LittleEndian.Uint16(data[9:11])
	pos := 11

	basePrice := getprice(data, &pos)
	preCloseDiff := getprice(data, &pos)
	openDiff := getprice(data, &pos)
	highDiff := getprice(data, &pos)
	lowDiff := getprice(data, &pos)
	serverTimeRaw := getprice(data, &pos)
	negPriceRaw := getprice(data, &pos)
	obj.reply.Vol = getprice(data, &pos)
	obj.reply.CurVol = getprice(data, &pos)
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	obj.reply.InVol = getprice(data, &pos)
	obj.reply.OutVol = getprice(data, &pos)
	obj.reply.SAmount = getprice(data, &pos)
	obj.reply.OpenAmount = getprice(data, &pos)

	for i := 0; i < 3; i++ {
		bid := getprice(data, &pos) + basePrice
		ask := getprice(data, &pos) + basePrice
		bidVol := getprice(data, &pos)
		askVol := getprice(data, &pos)
		obj.reply.BidLevels = append(obj.reply.BidLevels, Level{Price: float64(bid) / 100.0, Vol: bidVol})
		obj.reply.AskLevels = append(obj.reply.AskLevels, Level{Price: float64(ask) / 100.0, Vol: askVol})
	}

	obj.reply.Unknown = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2

	profilePrice := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		priceDelta := getprice(data, &pos)
		vol := getprice(data, &pos)
		buy := getprice(data, &pos)
		sell := getprice(data, &pos)
		profilePrice += priceDelta
		obj.reply.VolProfiles = append(obj.reply.VolProfiles, VolumeProfileItem{
			Price: float64(profilePrice) / 100.0,
			Vol:   vol,
			Buy:   buy,
			Sell:  sell,
		})
	}

	obj.reply.Close = float64(basePrice) / 100.0
	obj.reply.Open = float64(basePrice+openDiff) / 100.0
	obj.reply.High = float64(basePrice+highDiff) / 100.0
	obj.reply.Low = float64(basePrice+lowDiff) / 100.0
	obj.reply.PreClose = float64(basePrice+preCloseDiff) / 100.0
	obj.reply.ServerTime = formatServerTime(serverTimeRaw)
	obj.reply.NegPrice = float64(negPriceRaw) / 100.0

	return nil
}

func (obj *GetVolumeProfile) Reply() *GetVolumeProfileReply {
	return obj.reply
}
