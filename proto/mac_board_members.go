package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type MACBoardMembers struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardMembersRequest
	reply      *MACBoardMembersReply
}

type MACBoardMembersRequest struct {
	BoardCode uint32
	Reserved1 [9]byte
	SortType  uint16
	Start     uint32
	PageSize  uint8
	Zero      uint8
	SortOrder uint16
	Extra     [20]byte
}

type MACBoardMembersReply struct {
	Name   string
	Count  uint16
	Total  uint32
	Stocks []MACBoardMemberItem
}

type MACBoardMemberItem struct {
	Name   string
	Market uint16
	Symbol string
}

func NewMACBoardMembers() *MACBoardMembers {
	obj := &MACBoardMembers{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardMembersRequest),
		reply:      new(MACBoardMembersReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACBOARDMEMBERS
	obj.request.SortType = 14
	obj.request.PageSize = 80
	obj.request.SortOrder = 1
	return obj
}

func (obj *MACBoardMembers) SetParams(req *MACBoardMembersRequest) {
	if req.PageSize == 0 {
		req.PageSize = 80
	}
	if req.SortType == 0 {
		req.SortType = 14
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	obj.request = req
}

func (obj *MACBoardMembers) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_MACBOARDMEMBERS, payload.Bytes())
}

func (obj *MACBoardMembers) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 26 {
		return fmt.Errorf("invalid mac board members response length: %d", len(data))
	}

	obj.reply.Name = Utf8ToGbk(data[16:20])
	obj.reply.Total = binary.LittleEndian.Uint32(data[20:24])
	obj.reply.Count = binary.LittleEndian.Uint16(data[24:26])

	pos := 26
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+68 > len(data) {
			return fmt.Errorf("invalid mac board member item %d", i)
		}
		item := MACBoardMemberItem{
			Market: binary.LittleEndian.Uint16(data[pos : pos+2]),
			Symbol: Utf8ToGbk(data[pos+2 : pos+8]),
			Name:   Utf8ToGbk(data[pos+24 : pos+40]),
		}
		obj.reply.Stocks = append(obj.reply.Stocks, item)
		pos += 68
	}

	return nil
}

func (obj *MACBoardMembers) Reply() *MACBoardMembersReply {
	return obj.reply
}

type MACBoardMembersQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardMembersQuotesRequest
	reply      *MACBoardMembersQuotesReply
}

type MACBoardMembersQuotesRequest struct {
	BoardCode uint32
	Reserved1 [9]byte
	SortType  uint16
	Start     uint32
	PageSize  uint8
	Zero      uint8
	SortOrder uint8
	Extra     [21]byte
}

type MACBoardMembersQuotesReply struct {
	Name   string
	Count  uint16
	Total  uint32
	Stocks []MACBoardMemberQuoteItem
}

type MACBoardMemberQuoteItem struct {
	Name         string
	Market       uint16
	Symbol       string
	PreClose     float64
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Unknown6     float64
	VolumeRatio  float64
	Amount       float64
	TotalShares  float64
	FloatShares  float64
	EPS          float64
	ROE          float64
	Unknown13    float64
	MarketCap    float64
	PEDynamic    float64
	Zero16       float64
	Zero17       float64
	RiseSpeed    float64
	CurrentVol   uint16
	TurnoverRate float64
	Unknown21    float64
	Unknown22    float64
	LimitUp      float64
	LimitDown    float64
	Zero25       float64
	Unknown26    float64
	Unknown27    float64
	RiseSpeed2   float64
	Zero29       float64
	PEStatic     float64
	PETTM        float64
	Unknown31    float64
}

func NewMACBoardMembersQuotes() *MACBoardMembersQuotes {
	obj := &MACBoardMembersQuotes{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardMembersQuotesRequest),
		reply:      new(MACBoardMembersQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACBOARDMEMBERS
	obj.request.SortType = 14
	obj.request.PageSize = 80
	obj.request.SortOrder = 1
	obj.request.Extra = [21]byte{
		0x00, 0xff, 0xfc, 0xe1, 0xcc, 0x3f, 0x08, 0x03, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00,
	}
	return obj
}

func (obj *MACBoardMembersQuotes) SetParams(req *MACBoardMembersQuotesRequest) {
	if req.PageSize == 0 {
		req.PageSize = 80
	}
	if req.SortType == 0 {
		req.SortType = 14
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	if req.Extra == ([21]byte{}) {
		req.Extra = obj.request.Extra
	}
	obj.request = req
}

func (obj *MACBoardMembersQuotes) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request.BoardCode); err != nil {
		return nil, err
	}
	if _, err := payload.Write(obj.request.Reserved1[:]); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.SortType); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.Start); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.PageSize); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.Zero); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.SortOrder); err != nil {
		return nil, err
	}
	if _, err := payload.Write(obj.request.Extra[:]); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_MACBOARDMEMBERS, payload.Bytes())
}

func (obj *MACBoardMembersQuotes) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 26 {
		return fmt.Errorf("invalid mac board members quotes response length: %d", len(data))
	}

	obj.reply.Name = Utf8ToGbk(data[16:20])
	obj.reply.Total = binary.LittleEndian.Uint32(data[20:24])
	obj.reply.Count = binary.LittleEndian.Uint16(data[24:26])

	pos := 26
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+196 > len(data) {
			return fmt.Errorf("invalid mac board member quote item %d", i)
		}
		item := MACBoardMemberQuoteItem{
			Market: binary.LittleEndian.Uint16(data[pos : pos+2]),
			Symbol: Utf8ToGbk(data[pos+2 : pos+8]),
			Name:   Utf8ToGbk(data[pos+24 : pos+48]),
		}

		metrics := data[pos+68 : pos+196]
		floatAt := func(index int) float64 {
			return float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[index*4 : index*4+4])))
		}

		item.PreClose = floatAt(0)
		item.Open = floatAt(1)
		item.High = floatAt(2)
		item.Low = floatAt(3)
		item.Close = floatAt(4)
		item.Unknown6 = floatAt(5)
		item.VolumeRatio = floatAt(6)
		item.Amount = floatAt(7)
		item.TotalShares = floatAt(8)
		item.FloatShares = floatAt(9)
		item.EPS = floatAt(10)
		item.ROE = floatAt(11)
		item.Unknown13 = floatAt(12)
		item.MarketCap = floatAt(13)
		item.PEDynamic = floatAt(14)
		item.Zero16 = floatAt(15)
		item.Zero17 = floatAt(16)
		item.RiseSpeed = floatAt(17)
		item.CurrentVol = binary.LittleEndian.Uint16(metrics[72:74])
		offset := 76
		item.TurnoverRate = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.Unknown21 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.Unknown22 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.LimitUp = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.LimitDown = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.Zero25 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.Unknown26 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.Unknown27 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.RiseSpeed2 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.Zero29 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.PEStatic = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.PETTM = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))
		offset += 4
		item.Unknown31 = float64(math.Float32frombits(binary.LittleEndian.Uint32(metrics[offset : offset+4])))

		obj.reply.Stocks = append(obj.reply.Stocks, item)
		pos += 196
	}

	return nil
}

func (obj *MACBoardMembersQuotes) Reply() *MACBoardMembersQuotesReply {
	return obj.reply
}
