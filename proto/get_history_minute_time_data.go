package proto

import (
	"bytes"
	"encoding/binary"
)

type GetHistoryMinuteTimeData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryMinuteTimeDataRequest
	reply      *GetHistoryMinuteTimeDataReply
}

type GetHistoryMinuteTimeDataRequest struct {
	Date   int32
	Market uint8
	Code   [6]byte
}

type GetHistoryMinuteTimeDataReply struct {
	Count uint16
	List  []HistoryMinuteTimeData
}

type HistoryMinuteTimeData struct {
	Price float64
	Avg   float64
	Vol   int
}

func NewGetHistoryMinuteTimeData() *GetHistoryMinuteTimeData {
	obj := new(GetHistoryMinuteTimeData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetHistoryMinuteTimeDataRequest)
	obj.reply = new(GetHistoryMinuteTimeDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_HISTORYMINUTETIMEDATE
	return obj
}

func (obj *GetHistoryMinuteTimeData) SetParams(req *GetHistoryMinuteTimeDataRequest) {
	if req.Date > 0 {
		req.Date = -req.Date
	}
	obj.request = req
}

func (obj *GetHistoryMinuteTimeData) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0d
	obj.reqHeader.PkgLen2 = 0x0d

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetHistoryMinuteTimeData) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 10 // count + 2 unknown uint32 fields

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

		obj.reply.List = append(obj.reply.List, HistoryMinuteTimeData{
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

func (obj *GetHistoryMinuteTimeData) Reply() *GetHistoryMinuteTimeDataReply {
	return obj.reply
}
