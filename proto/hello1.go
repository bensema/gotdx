package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

type Hello1 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *Hello1Request
	reply      *Hello1Reply

	contentHex string
}

type Hello1Request struct {
}

type Hello1Reply struct {
	Info       string
	serverTime string
}

func NewHello1() *Hello1 {
	obj := new(Hello1)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(Hello1Request)
	obj.reply = new(Hello1Reply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_CMD1
	obj.contentHex = "01"
	return obj
}

func (obj *Hello1) Serialize() ([]byte, error) {
	b, err := hex.DecodeString(obj.contentHex)

	obj.reqHeader.PkgLen1 = 2 + uint16(len(b))
	obj.reqHeader.PkgLen2 = 2 + uint16(len(b))

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, obj.reqHeader)

	buf.Write(b)
	return buf.Bytes(), err
}

/*
00e60708051 50 f0 00 d3 a02b2020c03840384038403840384033a02b2020c0384038403840384038403 00 5a8a3401 f94a0100 5a8a3401 fd4a0100ff00e 700000101013f

	分  时    秒                                                                      日期
*/
func (obj *Hello1) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	serverInfo := Utf8ToGbk(data[68:])

	obj.reply.Info = serverInfo
	return nil
}

func (obj *Hello1) Reply() *Hello1Reply {
	return obj.reply
}
