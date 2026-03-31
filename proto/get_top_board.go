package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

type GetTopBoard struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetTopBoardRequest
	reply      *GetTopBoardReply
}

type GetTopBoardRequest struct {
	Category uint8
	Mode     uint8
	Reserved [7]byte
	Size     uint8
}

type GetTopBoardReply struct {
	Size               uint8
	Increase           []TopBoardItem
	Decrease           []TopBoardItem
	Amplitude          []TopBoardItem
	RiseSpeed          []TopBoardItem
	FallSpeed          []TopBoardItem
	VolRatio           []TopBoardItem
	PosCommissionRatio []TopBoardItem
	NegCommissionRatio []TopBoardItem
	Turnover           []TopBoardItem
}

type TopBoardItem struct {
	Market uint8
	Code   string
	Price  float64
	Value  float64
}

func NewGetTopBoard() *GetTopBoard {
	obj := new(GetTopBoard)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetTopBoardRequest)
	obj.reply = new(GetTopBoardReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_TOPBOARD
	obj.request.Mode = 5
	copy(obj.request.Reserved[:], []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00})
	return obj
}

func (obj *GetTopBoard) SetParams(req *GetTopBoardRequest) {
	if req.Mode == 0 {
		req.Mode = 5
	}
	if req.Size == 0 {
		req.Size = 20
	}
	if req.Reserved == [7]byte{} {
		req.Reserved = obj.request.Reserved
	}
	obj.request = req
}

func (obj *GetTopBoard) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0c
	obj.reqHeader.PkgLen2 = 0x0c

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetTopBoard) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	if err := binary.Read(bytes.NewBuffer(data[:1]), binary.LittleEndian, &obj.reply.Size); err != nil {
		return err
	}

	pos := 1
	targets := []*[]TopBoardItem{
		&obj.reply.Increase,
		&obj.reply.Decrease,
		&obj.reply.Amplitude,
		&obj.reply.RiseSpeed,
		&obj.reply.FallSpeed,
		&obj.reply.VolRatio,
		&obj.reply.PosCommissionRatio,
		&obj.reply.NegCommissionRatio,
		&obj.reply.Turnover,
	}

	for _, target := range targets {
		for i := uint8(0); i < obj.reply.Size; i++ {
			if pos+15 > len(data) {
				return io.ErrUnexpectedEOF
			}
			item := TopBoardItem{
				Market: data[pos],
				Code:   Utf8ToGbk(data[pos+1 : pos+7]),
				Price:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+7 : pos+11]))),
				Value:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+11 : pos+15]))),
			}
			*target = append(*target, item)
			pos += 15
		}
	}

	return nil
}

func (obj *GetTopBoard) Reply() *GetTopBoardReply {
	return obj.reply
}
