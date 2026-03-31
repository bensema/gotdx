package proto

import (
	"bytes"
	"encoding/binary"
)

type GetMinuteTimeData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetMinuteTimeDataRequest
	reply      *GetMinuteTimeDataReply
}

type GetMinuteTimeDataRequest struct {
	Market uint16
	Code   [6]byte
	Start  uint16
	Count  uint16
}

type GetMinuteTimeDataReply struct {
	Count uint16
	List  []MinuteTimeData
}

type MinuteTimeData struct {
	Price float64
	Avg   float64
	Vol   int
}

func NewGetMinuteTimeData() *GetMinuteTimeData {
	obj := new(GetMinuteTimeData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetMinuteTimeDataRequest)
	obj.reply = new(GetMinuteTimeDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_MINUTETIMEDATA
	return obj
}

func (obj *GetMinuteTimeData) SetParams(req *GetMinuteTimeDataRequest) {
	obj.request = req
}

func (obj *GetMinuteTimeData) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0e
	obj.reqHeader.PkgLen2 = 0x0e

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetMinuteTimeData) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	var ignored uint16
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ignored); err != nil {
		return err
	}
	pos += 2

	startPrice := 0
	startAvg := 0
	for index := uint16(0); index < obj.reply.Count; index++ {
		price := getprice(data, &pos)
		avg := getprice(data, &pos)
		vol := getprice(data, &pos)

		if startPrice != 0 {
			price += startPrice
		}
		if startAvg != 0 {
			avg += startAvg
		}

		obj.reply.List = append(obj.reply.List, MinuteTimeData{
			Price: float64(price) / 100.0,
			Avg:   float64(avg) / 10000.0,
			Vol:   vol,
		})

		if startPrice == 0 {
			startPrice = price
		}
		if startAvg == 0 {
			startAvg = avg
		}
	}

	return nil
}

func (obj *GetMinuteTimeData) Reply() *GetMinuteTimeDataReply {
	return obj.reply
}
