package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

type GetChartSampling struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetChartSamplingRequest
	reply      *GetChartSamplingReply
}

type GetChartSamplingRequest struct {
	Market   uint16
	Code     [6]byte
	Reserved [28]byte
}

type GetChartSamplingReply struct {
	Market   uint16
	Code     string
	Count    uint16
	PreClose float64
	Prices   []float64
}

func NewGetChartSampling() *GetChartSampling {
	obj := new(GetChartSampling)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetChartSamplingRequest)
	obj.reply = new(GetChartSamplingReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_CHARTSAMPLING
	copy(obj.request.Reserved[:], []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00,
	})
	return obj
}

func (obj *GetChartSampling) SetParams(req *GetChartSamplingRequest) {
	if req.Reserved == [28]byte{} {
		req.Reserved = obj.request.Reserved
	}
	obj.request = req
}

func (obj *GetChartSampling) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x26
	obj.reqHeader.PkgLen2 = 0x26

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetChartSampling) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	if len(data) < 42 {
		return io.ErrUnexpectedEOF
	}

	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:8])
	obj.reply.Count = binary.LittleEndian.Uint16(data[34:36])
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[36:40])))

	pos := 42
	for i := uint16(0); i < obj.reply.Count && pos+4 <= len(data); i++ {
		obj.reply.Prices = append(obj.reply.Prices, float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos:pos+4]))))
		pos += 4
	}

	return nil
}

func (obj *GetChartSampling) Reply() *GetChartSamplingReply {
	return obj.reply
}
