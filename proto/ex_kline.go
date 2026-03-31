package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type ExGetKLine struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetKLineRequest
	reply      *ExGetKLineReply
}

type ExGetKLineRequest struct {
	Category uint8
	Code     [9]byte
	Period   uint16
	Times    uint16
	Start    uint32
	Count    uint16
}

type ExGetKLineReply struct {
	Category uint8
	Name     string
	Period   uint16
	Times    uint16
	Start    uint32
	Count    uint16
	List     []ExKLineItem
}

type ExKLineItem struct {
	DateTime string
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Amount   float64
	Vol      uint32
}

func NewExGetKLine() *ExGetKLine {
	obj := &ExGetKLine{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetKLineRequest),
		reply:      new(ExGetKLineReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXKLINE
	return obj
}

func (obj *ExGetKLine) SetParams(req *ExGetKLineRequest) {
	if req.Times == 0 {
		req.Times = 1
	}
	obj.request = req
}

func (obj *ExGetKLine) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXKLINE, payload.Bytes())
}

func (obj *ExGetKLine) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 20 {
		return fmt.Errorf("invalid ex kline response length: %d", len(data))
	}

	obj.reply.Category = data[0]
	obj.reply.Name = Utf8ToGbk(data[1:10])
	obj.reply.Period = binary.LittleEndian.Uint16(data[10:12])
	obj.reply.Times = binary.LittleEndian.Uint16(data[12:14])
	obj.reply.Start = binary.LittleEndian.Uint32(data[14:18])
	obj.reply.Count = binary.LittleEndian.Uint16(data[18:20])
	pos := 20

	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+32 > len(data) {
			return fmt.Errorf("invalid ex kline item %d", i)
		}
		dateNum := binary.LittleEndian.Uint32(data[pos : pos+4])
		ts, ok := decodeDateNum(obj.reply.Period, dateNum)
		if !ok {
			return fmt.Errorf("invalid ex kline datetime: %d", dateNum)
		}
		item := ExKLineItem{
			DateTime: ts.Format("2006-01-02 15:04:05"),
			Open:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))),
			High:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))),
			Low:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))),
			Close:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+16 : pos+20]))),
			Amount:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+20 : pos+24]))),
			Vol:      binary.LittleEndian.Uint32(data[pos+24 : pos+28]),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 32
	}
	return nil
}

func (obj *ExGetKLine) Reply() *ExGetKLineReply {
	return obj.reply
}

type ExGetHistoryTransaction struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetHistoryTransactionRequest
	reply      *ExGetHistoryTransactionReply
}

type ExGetHistoryTransactionRequest struct {
	Date     uint32
	Category uint8
	Code     [43]byte
	Count    uint16
}

type ExGetHistoryTransactionReply struct {
	Category   uint8
	Name       string
	Date       string
	StartPrice float64
	Count      uint16
	List       []ExHistoryTransactionItem
}

type ExHistoryTransactionItem struct {
	Time   string
	Price  uint32
	Vol    uint32
	Action string
}

func NewExGetHistoryTransaction() *ExGetHistoryTransaction {
	obj := &ExGetHistoryTransaction{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetHistoryTransactionRequest),
		reply:      new(ExGetHistoryTransactionReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXHISTORYTRANSACTION
	obj.request.Count = 0x78
	return obj
}

func (obj *ExGetHistoryTransaction) SetParams(req *ExGetHistoryTransactionRequest) {
	if req.Count == 0 {
		req.Count = 0x78
	}
	obj.request = req
}

func (obj *ExGetHistoryTransaction) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXHISTORYTRANSACTION, payload.Bytes())
}

func (obj *ExGetHistoryTransaction) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 58 {
		return fmt.Errorf("invalid ex history transaction response length: %d", len(data))
	}

	obj.reply.Category = data[0]
	obj.reply.Name = Utf8ToGbk(data[1:40])
	dateRaw := binary.LittleEndian.Uint32(data[40:44])
	obj.reply.Date = fmt.Sprintf("%04d-%02d-%02d", dateRaw/10000, (dateRaw%10000)/100, dateRaw%100)
	obj.reply.StartPrice = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[44:48])))
	obj.reply.Count = binary.LittleEndian.Uint16(data[56:58])
	pos := 58

	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+16 > len(data) {
			return fmt.Errorf("invalid ex history transaction item %d", i)
		}
		minutes := binary.LittleEndian.Uint16(data[pos : pos+2])
		actionCode := binary.LittleEndian.Uint16(data[pos+14 : pos+16])
		item := ExHistoryTransactionItem{
			Time:   fmt.Sprintf("%02d:%02d", (minutes/60)%24, minutes%60),
			Price:  binary.LittleEndian.Uint32(data[pos+2 : pos+6]),
			Vol:    binary.LittleEndian.Uint32(data[pos+6 : pos+10]),
			Action: exAction(actionCode),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 16
	}
	return nil
}

func (obj *ExGetHistoryTransaction) Reply() *ExGetHistoryTransactionReply {
	return obj.reply
}

func exAction(code uint16) string {
	switch code {
	case 0:
		return "BUY"
	case 1:
		return "SELL"
	case 2:
		return "NEUTRAL"
	default:
		return ""
	}
}
