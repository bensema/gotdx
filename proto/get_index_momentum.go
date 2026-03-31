package proto

import (
	"bytes"
	"encoding/binary"
)

type GetIndexMomentum struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetIndexMomentumRequest
	reply      *GetIndexMomentumReply
}

type GetIndexMomentumRequest struct {
	Market uint16
	Code   [6]byte
}

type GetIndexMomentumReply struct {
	Count  uint16
	Values []int
}

func NewGetIndexMomentum() *GetIndexMomentum {
	obj := new(GetIndexMomentum)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetIndexMomentumRequest)
	obj.reply = new(GetIndexMomentumReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_INDEXMOMENTUM
	return obj
}

func (obj *GetIndexMomentum) SetParams(req *GetIndexMomentumRequest) {
	obj.request = req
}

func (obj *GetIndexMomentum) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0a
	obj.reqHeader.PkgLen2 = 0x0a

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetIndexMomentum) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	if err := binary.Read(bytes.NewBuffer(data[:2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2

	startMomentum := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		momentum := getprice(data, &pos)
		startMomentum += momentum
		obj.reply.Values = append(obj.reply.Values, startMomentum)
	}

	return nil
}

func (obj *GetIndexMomentum) Reply() *GetIndexMomentumReply {
	return obj.reply
}
