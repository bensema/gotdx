package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

type GetHistoryOrders struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryOrdersRequest
	reply      *GetHistoryOrdersReply
}

type GetHistoryOrdersRequest struct {
	Date   uint32
	Market uint8
	Code   [6]byte
}

type GetHistoryOrdersReply struct {
	Count    uint16
	PreClose float64
	List     []HistoryOrderData
}

type HistoryOrderData struct {
	Price   float64
	Unknown int
	Vol     int
}

func NewGetHistoryOrders() *GetHistoryOrders {
	obj := new(GetHistoryOrders)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetHistoryOrdersRequest)
	obj.reply = new(GetHistoryOrdersReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_HISTORYORDERS
	return obj
}

func (obj *GetHistoryOrders) SetParams(req *GetHistoryOrdersRequest) {
	obj.request = req
}

func (obj *GetHistoryOrders) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0d
	obj.reqHeader.PkgLen2 = 0x0d

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetHistoryOrders) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4

	lastPrice := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		priceRaw := getprice(data, &pos)
		unknown := getprice(data, &pos)
		vol := getprice(data, &pos)
		lastPrice += priceRaw
		obj.reply.List = append(obj.reply.List, HistoryOrderData{
			Price:   float64(lastPrice) / 100.0,
			Unknown: unknown,
			Vol:     vol,
		})
	}

	return nil
}

func (obj *GetHistoryOrders) Reply() *GetHistoryOrdersReply {
	return obj.reply
}
