package proto

import (
	"bytes"
	"encoding/binary"
)

type GetSecurityCount struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityCountRequest
	reply      *GetSecurityCountReply
	contentHex string
}

type GetSecurityCountRequest struct {
	Market uint16
	Date   uint32
}

type GetSecurityCountReply struct {
	Count uint16
}

func NewGetSecurityCount() *GetSecurityCount {
	obj := new(GetSecurityCount)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityCountRequest)
	obj.reply = new(GetSecurityCountReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYCOUNT
	return obj
}

func (obj *GetSecurityCount) SetParams(req *GetSecurityCountRequest) {
	if req.Date == 0 {
		req.Date = todayDate()
	}
	obj.request = req
}

func (obj *GetSecurityCount) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2 + 2 + 4
	obj.reqHeader.PkgLen2 = 2 + 2 + 4

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetSecurityCount) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	return nil
}

func (obj *GetSecurityCount) Reply() *GetSecurityCountReply {
	return obj.reply
}
