package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

type ExStock struct {
	Category uint8
	Code     string
}

type ExQuoteItem struct {
	Category      uint8
	Code          string
	Active        uint32
	PreClose      float64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	OpenPosition  int
	AddPosition   int
	Vol           int
	CurVol        int
	Amount        float64
	InVol         int
	OutVol        int
	HoldPosition  int
	Unknown14     int
	BidLevels     []Level
	AskLevels     []Level
	Settlement    float64
	Avg           float64
	PreSettlement float64
	PreVol        float64
	Day3Raise     float64
	Settlement2   float64
	Date          string
	RaiseSpeed    float64
	Unknown1      uint16
	Unknown2      uint32
	Unknown3      []uint32
	Unknown7      float64
	Unknown8      float64
	Unknown9      uint32
	Unknown10     float64
	Unknown11     uint16
	Unknown12     uint8
}

type ExGetQuotesList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuotesListRequest
	reply      *ExGetQuotesListReply
}

type ExGetQuotesListRequest struct {
	Category    uint8
	SortType    uint16
	Start       uint16
	Count       uint16
	SortReverse uint16
}

type ExGetQuotesListReply struct {
	Count uint16
	List  []ExQuoteItem
}

func NewExGetQuotesList() *ExGetQuotesList {
	obj := &ExGetQuotesList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuotesListRequest),
		reply:      new(ExGetQuotesListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTESLIST
	return obj
}

func (obj *ExGetQuotesList) SetParams(req *ExGetQuotesListRequest) {
	obj.request = req
}

func (obj *ExGetQuotesList) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXQUOTESLIST, payload.Bytes())
}

func (obj *ExGetQuotesList) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	return parseExQuotesReply(data, &obj.reply.Count, &obj.reply.List)
}

func (obj *ExGetQuotesList) Reply() *ExGetQuotesListReply {
	return obj.reply
}

type ExGetQuote struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuoteRequest
	reply      *ExGetQuoteReply
}

type ExGetQuoteRequest struct {
	Category uint8
	Code     [9]byte
}

type ExGetQuoteReply struct {
	Item ExQuoteItem
}

func NewExGetQuote() *ExGetQuote {
	obj := &ExGetQuote{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuoteRequest),
		reply:      new(ExGetQuoteReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTESINGLE
	return obj
}

func (obj *ExGetQuote) SetParams(req *ExGetQuoteRequest) {
	obj.request = req
}

func (obj *ExGetQuote) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXQUOTESINGLE, payload.Bytes())
}

func (obj *ExGetQuote) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	item, err := parseExQuoteItem(data, 9)
	if err != nil {
		return err
	}
	obj.reply.Item = item
	return nil
}

func (obj *ExGetQuote) Reply() *ExGetQuoteReply {
	return obj.reply
}

type ExGetQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuotesRequest
	reply      *ExGetQuotesReply
}

type ExGetQuotesRequest struct {
	Stocks []ExStock
}

type ExGetQuotesReply struct {
	Count uint16
	List  []ExQuoteItem
}

func NewExGetQuotes() *ExGetQuotes {
	obj := &ExGetQuotes{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuotesRequest),
		reply:      new(ExGetQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES
	return obj
}

func (obj *ExGetQuotes) SetParams(req *ExGetQuotesRequest) {
	obj.request = req
}

func (obj *ExGetQuotes) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, uint8(5)); err != nil {
		return nil, err
	}
	if _, err := payload.Write(make([]byte, 7)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(len(obj.request.Stocks))); err != nil {
		return nil, err
	}
	for _, stock := range obj.request.Stocks {
		if err := binary.Write(payload, binary.LittleEndian, stock.Category); err != nil {
			return nil, err
		}
		code := make([]byte, 23)
		copy(code, stock.Code)
		if _, err := payload.Write(code); err != nil {
			return nil, err
		}
	}
	return serializeExRequest(KMSG_EXQUOTES, payload.Bytes())
}

func (obj *ExGetQuotes) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	return parseExQuotesReply(data, &obj.reply.Count, &obj.reply.List)
}

func (obj *ExGetQuotes) Reply() *ExGetQuotesReply {
	return obj.reply
}

type ExGetQuotes2 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetQuotesRequest
	reply      *ExGetQuotesReply
}

func NewExGetQuotes2() *ExGetQuotes2 {
	obj := &ExGetQuotes2{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetQuotesRequest),
		reply:      new(ExGetQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES2
	return obj
}

func (obj *ExGetQuotes2) SetParams(req *ExGetQuotesRequest) {
	obj.request = req
}

func (obj *ExGetQuotes2) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, uint16(2)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(3148)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(600)); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(len(obj.request.Stocks))); err != nil {
		return nil, err
	}
	for _, stock := range obj.request.Stocks {
		if err := binary.Write(payload, binary.LittleEndian, stock.Category); err != nil {
			return nil, err
		}
		code := make([]byte, 23)
		copy(code, stock.Code)
		if _, err := payload.Write(code); err != nil {
			return nil, err
		}
	}
	return serializeExRequest(KMSG_EXQUOTES2, payload.Bytes())
}

func (obj *ExGetQuotes2) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	return parseExQuotesReply(data, &obj.reply.Count, &obj.reply.List)
}

func (obj *ExGetQuotes2) Reply() *ExGetQuotesReply {
	return obj.reply
}

func parseExQuotesReply(data []byte, count *uint16, out *[]ExQuoteItem) error {
	if len(data) < 10 {
		return fmt.Errorf("invalid ex quotes response length: %d", len(data))
	}
	*count = binary.LittleEndian.Uint16(data[8:10])
	for i := uint16(0); i < *count; i++ {
		base := 10 + int(i)*314
		if base+314 > len(data) {
			return fmt.Errorf("invalid ex quotes item %d", i)
		}
		item, err := parseExQuoteItem(data[base:base+314], 23)
		if err != nil {
			return err
		}
		*out = append(*out, item)
	}
	return nil
}

func parseExQuoteItem(data []byte, codeLen int) (ExQuoteItem, error) {
	minLen := 291 + codeLen
	if len(data) < minLen {
		return ExQuoteItem{}, fmt.Errorf("invalid ex quote item length: %d", len(data))
	}

	item := ExQuoteItem{
		Category: data[0],
		Code:     Utf8ToGbk(data[1 : 1+codeLen]),
	}
	pos := 1 + codeLen

	item.Active = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.OpenPosition = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.AddPosition = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.Vol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.CurVol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.InVol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.OutVol = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.Unknown14 = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	item.HoldPosition = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4

	for i := 0; i < 5; i++ {
		price := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4])))
		item.BidLevels = append(item.BidLevels, Level{Price: price})
	}
	pos += 20
	for i := 0; i < 5; i++ {
		item.BidLevels[i].Vol = int(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4]))
	}
	pos += 20
	for i := 0; i < 5; i++ {
		price := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4])))
		item.AskLevels = append(item.AskLevels, Level{Price: price})
	}
	pos += 20
	for i := 0; i < 5; i++ {
		item.AskLevels[i].Vol = int(binary.LittleEndian.Uint32(data[pos+i*4 : pos+i*4+4]))
	}
	pos += 20

	item.Unknown1 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2
	item.Settlement = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Unknown2 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.Avg = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.PreSettlement = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Unknown3 = append(item.Unknown3,
		binary.LittleEndian.Uint32(data[pos:pos+4]),
		binary.LittleEndian.Uint32(data[pos+4:pos+8]),
		binary.LittleEndian.Uint32(data[pos+8:pos+12]),
		binary.LittleEndian.Uint32(data[pos+12:pos+16]),
	)
	pos += 16
	item.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4

	pos += 12
	item.PreVol = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Unknown7 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	pos += 12
	item.Unknown8 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Day3Raise = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	pos += 25
	item.Settlement2 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	dateRaw := binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.Unknown9 = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	item.RaiseSpeed = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	item.Unknown10 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	pos += 24
	item.Unknown11 = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2
	item.Unknown12 = data[pos]

	if dateRaw == 0 {
		item.Date = "1900-01-01"
	} else {
		d := time.Date(int(dateRaw/10000), time.Month((dateRaw%10000)/100), int(dateRaw%100), 0, 0, 0, 0, time.Local)
		item.Date = d.Format("2006-01-02")
	}

	return item, nil
}
