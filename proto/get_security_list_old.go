package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type GetSecurityListOld struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityListOldRequest
	reply      *GetSecurityListOldReply
}

type GetSecurityListOldRequest struct {
	Market uint16
	Start  uint16
}

type GetSecurityListOldReply struct {
	Count uint16
	List  []Security
}

func NewGetSecurityListOld() *GetSecurityListOld {
	obj := &GetSecurityListOld{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(GetSecurityListOldRequest),
		reply:      new(GetSecurityListOldReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYLIST_OLD
	return obj
}

func (obj *GetSecurityListOld) SetParams(req *GetSecurityListOldRequest) {
	obj.request = req
}

func (obj *GetSecurityListOld) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 6
	obj.reqHeader.PkgLen2 = 6
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	err := binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetSecurityListOld) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 2 {
		return fmt.Errorf("invalid security list old response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+29 > len(data) {
			return fmt.Errorf("invalid security list old item %d", i)
		}
		item := Security{
			Code:     Utf8ToGbk(data[pos : pos+6]),
			Vol:      binary.LittleEndian.Uint16(data[pos+6 : pos+8]),
			Name:     Utf8ToGbk(data[pos+8 : pos+16]),
			Unknown2: binary.LittleEndian.Uint16(data[pos+25 : pos+27]),
			Unknown3: binary.LittleEndian.Uint16(data[pos+27 : pos+29]),
		}
		item.VolUnit = item.Vol
		item.DecimalPoint = int8(data[pos+20])
		item.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+21 : pos+25])))
		obj.reply.List = append(obj.reply.List, item)
		pos += 29
	}
	return nil
}

func (obj *GetSecurityListOld) Reply() *GetSecurityListOldReply {
	return obj.reply
}
