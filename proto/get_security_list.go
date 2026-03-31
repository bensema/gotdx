package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

type GetSecurityList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityListRequest
	reply      *GetSecurityListReply

	contentHex string
}

type GetSecurityListRequest struct {
	Market uint16
	Start  uint32
	Count  uint32
	Zero   uint32
}

type GetSecurityListReply struct {
	Count uint16
	List  []Security
}

type Security struct {
	Code         string
	Vol          uint16
	VolUnit      uint16
	DecimalPoint int8
	Name         string
	PreClose     float64
	Unknown1     float32
	Unknown2     uint16
	Unknown3     uint16
}

func NewGetSecurityList() *GetSecurityList {
	obj := new(GetSecurityList)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityListRequest)
	obj.reply = new(GetSecurityListReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYLIST
	return obj
}

func (obj *GetSecurityList) SetParams(req *GetSecurityListRequest) {
	if req.Count == 0 {
		req.Count = 1600
	}
	obj.request = req
}

func (obj *GetSecurityList) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2 + 2 + 4 + 4 + 4
	obj.reqHeader.PkgLen2 = 2 + 2 + 4 + 4 + 4

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)

	return buf.Bytes(), err
}

func (obj *GetSecurityList) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	pos += 2
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := Security{}
		var code [6]byte
		binary.Read(bytes.NewBuffer(data[pos:pos+6]), binary.LittleEndian, &code)
		pos += 6
		ele.Code = string(code[:])

		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Vol)
		pos += 2
		ele.VolUnit = ele.Vol

		var name [16]byte
		binary.Read(bytes.NewBuffer(data[pos:pos+16]), binary.LittleEndian, &name)
		pos += 16

		ele.Name = Utf8ToGbk(name[:])

		var unknown1 uint32
		binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &unknown1)
		ele.Unknown1 = math.Float32frombits(unknown1)
		pos += 4
		binary.Read(bytes.NewBuffer(data[pos:pos+1]), binary.LittleEndian, &ele.DecimalPoint)
		pos += 1
		ele.PreClose = getfloat32(data, &pos)

		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Unknown2)
		pos += 2
		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Unknown3)
		pos += 2

		obj.reply.List = append(obj.reply.List, ele)
	}
	return err
}

func (obj *GetSecurityList) Reply() *GetSecurityListReply {
	return obj.reply
}
