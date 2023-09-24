package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type GetTransactionData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetTransactionDataRequest
	reply      *GetTransactionDataReply

	contentHex string
}

type GetTransactionDataRequest struct {
	Market uint16
	Code   [6]byte
	Start  uint16
	Count  uint16
}

type GetTransactionDataReply struct {
	Count uint16
	List  []TransactionData
}

type TransactionData struct {
	Time      string
	Price     float64
	Vol       int
	Num       int
	BuyOrSell int
}

func NewGetTransactionData() *GetTransactionData {
	obj := new(GetTransactionData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetTransactionDataRequest)
	obj.reply = new(GetTransactionDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	//obj.reqHeader.PkgLen1  =
	//obj.reqHeader.PkgLen2  =
	obj.reqHeader.Method = KMSG_TRANSACTIONDATA
	//obj.reqHeader.Method = KMSG_MINUTETIMEDATA
	obj.contentHex = ""
	return obj
}

func (obj *GetTransactionData) SetParams(req *GetTransactionDataRequest) {
	obj.request = req
}

func (obj *GetTransactionData) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0e
	obj.reqHeader.PkgLen2 = 0x0e

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

func (obj *GetTransactionData) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	pos += 2

	lastprice := 0
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := TransactionData{}
		hour, minute := gettime(data, &pos)
		ele.Time = fmt.Sprintf("%02d:%02d", hour, minute)
		priceraw := getprice(data, &pos)
		ele.Vol = getprice(data, &pos)
		ele.Num = getprice(data, &pos)
		ele.BuyOrSell = getprice(data, &pos)
		lastprice += priceraw
		ele.Price = float64(lastprice) / baseUnit(string(obj.request.Code[:]))
		_ = getprice(data, &pos)
		obj.reply.List = append(obj.reply.List, ele)
	}
	return err
}

func (obj *GetTransactionData) Reply() *GetTransactionDataReply {
	return obj.reply
}
