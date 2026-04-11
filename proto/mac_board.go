package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type MACBoardListRequest struct {
	PageSize  uint16
	BoardType uint16
	SortType  uint8
	SortOrder uint8
	Start     uint16
	One       uint16
	Reserved  [8]byte
}

type MACBoardCount struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardListRequest
	reply      *MACBoardCountReply
}

type MACBoardCountReply struct {
	CountAll uint16
	Total    uint16
}

func NewMACBoardCount() *MACBoardCount {
	obj := &MACBoardCount{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardListRequest),
		reply:      new(MACBoardCountReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXBOARDLIST
	obj.request.PageSize = 150
	obj.request.SortOrder = 1
	obj.request.One = 1
	return obj
}

func (obj *MACBoardCount) SetParams(req *MACBoardListRequest) {
	if req.PageSize == 0 {
		req.PageSize = 150
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *MACBoardCount) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXBOARDLIST, payload.Bytes())
}

func (obj *MACBoardCount) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 4 {
		return fmt.Errorf("invalid mac board count response length: %d", len(data))
	}
	obj.reply.CountAll = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Total = binary.LittleEndian.Uint16(data[2:4])
	return nil
}

func (obj *MACBoardCount) Reply() *MACBoardCountReply {
	return obj.reply
}

type MACBoardList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardListRequest
	reply      *MACBoardListReply
}

type MACBoardListReply struct {
	CountAll uint16
	Total    uint16
	Count    uint16
	List     []MACBoardListItem
}

type MACBoardListItem struct {
	Market          uint16
	Code            string
	Name            string
	Price           float64
	RiseSpeed       float64
	PreClose        float64
	SymbolMarket    uint16
	SymbolCode      string
	SymbolName      string
	SymbolPrice     float64
	SymbolRiseSpeed float64
	SymbolPreClose  float64
}

func NewMACBoardList() *MACBoardList {
	obj := &MACBoardList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardListRequest),
		reply:      new(MACBoardListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXBOARDLIST
	obj.request.PageSize = 150
	obj.request.SortOrder = 1
	obj.request.One = 1
	return obj
}

func (obj *MACBoardList) SetParams(req *MACBoardListRequest) {
	if req.PageSize == 0 {
		req.PageSize = 150
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *MACBoardList) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXBOARDLIST, payload.Bytes())
}

func (obj *MACBoardList) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 4 {
		return fmt.Errorf("invalid mac board list response length: %d", len(data))
	}

	obj.reply.CountAll = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Total = binary.LittleEndian.Uint16(data[2:4])
	obj.reply.Count = obj.reply.CountAll / 2
	if obj.reply.Count == 0 && obj.reply.CountAll > 0 {
		obj.reply.Count = obj.reply.CountAll
	}

	pos := 4
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+160 > len(data) {
			return fmt.Errorf("invalid mac board list item %d", i)
		}
		item := MACBoardListItem{
			Market:          binary.LittleEndian.Uint16(data[pos : pos+2]),
			Code:            Utf8ToGbk(data[pos+2 : pos+8]),
			Name:            Utf8ToGbk(data[pos+24 : pos+68]),
			Price:           float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+68 : pos+72]))),
			RiseSpeed:       float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+72 : pos+76]))),
			PreClose:        float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+76 : pos+80]))),
			SymbolMarket:    binary.LittleEndian.Uint16(data[pos+80 : pos+82]),
			SymbolCode:      Utf8ToGbk(data[pos+82 : pos+88]),
			SymbolName:      Utf8ToGbk(data[pos+104 : pos+148]),
			SymbolPrice:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+148 : pos+152]))),
			SymbolRiseSpeed: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+152 : pos+156]))),
			SymbolPreClose:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+156 : pos+160]))),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 160
	}

	return nil
}

func (obj *MACBoardList) Reply() *MACBoardListReply {
	return obj.reply
}
