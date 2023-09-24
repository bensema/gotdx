package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type GetHistoryTransactionData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryTransactionDataRequest
	reply      *GetHistoryTransactionDataReply

	contentHex string
}

type GetHistoryTransactionDataRequest struct {
	Date   uint32
	Market uint16
	Code   [6]byte
	Start  uint16
	Count  uint16
}

type GetHistoryTransactionDataReply struct {
	Count uint16
	List  []HistoryTransactionData
}

type HistoryTransactionData struct {
	Time      string
	Price     float64
	Vol       int
	Num       int
	BuyOrSell int
}

func NewGetHistoryTransactionData() *GetHistoryTransactionData {
	obj := new(GetHistoryTransactionData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetHistoryTransactionDataRequest)
	obj.reply = new(GetHistoryTransactionDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	//obj.reqHeader.PkgLen1  =
	//obj.reqHeader.PkgLen2  =
	obj.reqHeader.Method = KMSG_HISTORYTRANSACTIONDATA
	//obj.reqHeader.Method = KMSG_MINUTETIMEDATA
	obj.contentHex = ""
	return obj
}

func (obj *GetHistoryTransactionData) SetParams(req *GetHistoryTransactionDataRequest) {
	obj.request = req
}

func (obj *GetHistoryTransactionData) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x12
	obj.reqHeader.PkgLen2 = 0x12

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	b, err := hex.DecodeString(obj.contentHex)
	buf.Write(b)

	//b, err := hex.DecodeString(obj.contentHex)
	//buf.Write(b)

	//err = binary.Write(buf, binary.LittleEndian, uint16(len(obj.stocks)))

	return buf.Bytes(), err
}

func (obj *GetHistoryTransactionData) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	// 跳过4个字节
	pos += 6

	lastprice := 0
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := HistoryTransactionData{}
		h, m := gettime(data, &pos)
		ele.Time = fmt.Sprintf("%02d:%02d", h, m)
		priceraw := getprice(data, &pos)
		ele.Vol = getprice(data, &pos)
		ele.BuyOrSell = getprice(data, &pos)
		getprice(data, &pos)

		lastprice = lastprice + priceraw
		ele.Price = float64(lastprice) / baseUnit(string(obj.request.Code[:]))
		obj.reply.List = append(obj.reply.List, ele)
	}
	return err
}

func (obj *GetHistoryTransactionData) Reply() *GetHistoryTransactionDataReply {
	return obj.reply
}
