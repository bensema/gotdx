package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type GetAuction struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetAuctionRequest
	reply      *GetAuctionReply
}

type GetAuctionRequest struct {
	Market uint16
	Code   [6]byte
	Zero1  uint32
	Mode   uint32
	Zero2  uint32
	Start  uint32
	Count  uint32
}

type GetAuctionReply struct {
	Count uint16
	List  []AuctionData
}

type AuctionData struct {
	Time      string
	Price     float64
	Matched   uint32
	Unmatched uint32
}

func NewGetAuction() *GetAuction {
	obj := new(GetAuction)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetAuctionRequest)
	obj.reply = new(GetAuctionReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_AUCTION
	obj.request.Mode = 3
	return obj
}

func (obj *GetAuction) SetParams(req *GetAuctionRequest) {
	if req.Mode == 0 {
		req.Mode = 3
	}
	if req.Count == 0 {
		req.Count = 500
	}
	obj.request = req
}

func (obj *GetAuction) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x1e
	obj.reqHeader.PkgLen2 = 0x1e

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetAuction) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	if err := binary.Read(bytes.NewBuffer(data[:2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}

	for i := uint16(0); i < obj.reply.Count; i++ {
		base := int(i)*16 + 2
		if base+16 > len(data) {
			return io.ErrUnexpectedEOF
		}

		timeRaw := binary.LittleEndian.Uint16(data[base : base+2])
		priceBits := binary.LittleEndian.Uint32(data[base+2 : base+6])
		matched := binary.LittleEndian.Uint32(data[base+6 : base+10])
		unmatched := binary.LittleEndian.Uint32(data[base+10 : base+14])
		second := data[base+15]

		obj.reply.List = append(obj.reply.List, AuctionData{
			Time:      fmt.Sprintf("%02d:%02d:%02d", timeRaw/60, timeRaw%60, second),
			Price:     float64(math.Float32frombits(priceBits)),
			Matched:   matched,
			Unmatched: unmatched,
		})
	}

	return nil
}

func (obj *GetAuction) Reply() *GetAuctionReply {
	return obj.reply
}
