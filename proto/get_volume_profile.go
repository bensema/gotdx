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
	Market uint16
	Code   [6]byte
}

type GetVolumeProfileReply struct {
	Count       uint16
	Market      uint8
	Code        string
	Active      uint16
	Close       float64
	Open        float64
	High        float64
	Low         float64
	PreClose    float64
	ServerTime  string
	NegPrice    float64
	Vol         int
	CurVol      int
	Amount      float64
	InVol       int
	OutVol      int
	SAmount     int
	OpenAmount  int
	BidLevels   []Level
	AskLevels   []Level
	Unknown     uint16
	VolProfiles []VolumeProfileItem
}

type VolumeProfileItem struct {
	Price float64
	Vol   int
	Buy   int
	Sell  int
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
