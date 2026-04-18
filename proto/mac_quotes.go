package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACQuotes 表示 0x122D MAC 行情快照协议。
type MACQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACQuotesRequest
	reply      *MACQuotesReply
}

// MACQuotesRequest 表示 0x122D MAC 行情快照请求。
type MACQuotesRequest struct {
	Market uint16
	Code   [22]byte
	Zero1  uint16
	Zero2  uint16
	One    uint16
	Zero3  uint16
	Zero4  uint16
	Zero5  uint16
	Zero6  uint16
}

// MACQuotesReply 表示 MAC 单只行情快照及分时采样响应。
type MACQuotesReply struct {
	Market    uint16
	Code      string
	Date      uint32
	Unknown   uint8
	Price     float64
	Count     uint16
	ChartData []MACQuoteChartItem

	Name     string
	Decimal  uint8
	Category uint16
	VolUnit  float64
	DateTime string
	PreClose float64
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Momentum float64
	Vol      uint32
	Amount   float64
	Turnover float64
	Avg      float64
	Industry uint32
}

// MACQuoteChartItem 表示 MAC 分时采样点。
type MACQuoteChartItem struct {
	Time     string
	Price    float64
	Avg      float64
	Vol      uint32
	Momentum float64
}

// NewMACQuotes 创建 MAC 行情快照协议对象。
func NewMACQuotes(req *MACQuotesRequest) *MACQuotes {
	obj := &MACQuotes{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACQuotesRequest),
		reply:      new(MACQuotesReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACQUOTES
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACQuotes) applyRequest(req *MACQuotesRequest) {
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *MACQuotes) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_MACQUOTES, payload.Bytes())
}

func (obj *MACQuotes) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 35 {
		return fmt.Errorf("invalid mac quotes response length: %d", len(data))
	}

	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:24])
	obj.reply.Date = binary.LittleEndian.Uint32(data[24:28])
	obj.reply.Unknown = data[28]
	obj.reply.Price = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[29:33])))
	obj.reply.Count = binary.LittleEndian.Uint16(data[33:35])

	pos := 35
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+18 > len(data) {
			return fmt.Errorf("invalid mac quote chart item %d", i)
		}
		minutes := binary.LittleEndian.Uint16(data[pos : pos+2])
		item := MACQuoteChartItem{
			Time:     time.Date(0, 1, 1, int(minutes/60)%24, int(minutes%60), 0, 0, time.Local).Format("15:04:05"),
			Price:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))),
			Avg:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+6 : pos+10]))),
			Vol:      binary.LittleEndian.Uint32(data[pos+10 : pos+14]),
			Momentum: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+14 : pos+18]))),
		}
		obj.reply.ChartData = append(obj.reply.ChartData, item)
		pos += 18
	}

	if pos+109 > len(data) {
		return fmt.Errorf("invalid mac quotes summary length: %d", len(data)-pos)
	}

	obj.reply.Name = Utf8ToGbk(data[pos : pos+44])
	obj.reply.Decimal = data[pos+44]
	obj.reply.Category = binary.LittleEndian.Uint16(data[pos+45 : pos+47])
	obj.reply.VolUnit = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+47 : pos+51])))
	dateRaw := binary.LittleEndian.Uint32(data[pos+56 : pos+60])
	timeRaw := binary.LittleEndian.Uint32(data[pos+60 : pos+64])
	obj.reply.DateTime = formatMACQuoteDateTime(dateRaw, timeRaw)
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+64 : pos+68])))
	obj.reply.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+68 : pos+72])))
	obj.reply.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+72 : pos+76])))
	obj.reply.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+76 : pos+80])))
	obj.reply.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+80 : pos+84])))
	obj.reply.Momentum = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+84 : pos+88])))
	obj.reply.Vol = binary.LittleEndian.Uint32(data[pos+88 : pos+92])
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+92 : pos+96])))
	obj.reply.Turnover = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+108 : pos+112])))
	obj.reply.Avg = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+112 : pos+116])))
	obj.reply.Industry = binary.LittleEndian.Uint32(data[pos+116 : pos+120])

	return nil
}

// Response 返回解析后的 MAC 行情快照响应。
func (obj *MACQuotes) Response() *MACQuotesReply {
	return obj.reply
}

func formatMACQuoteDateTime(dateRaw uint32, timeRaw uint32) string {
	year := int(dateRaw / 10000)
	month := int((dateRaw % 10000) / 100)
	day := int(dateRaw % 100)
	hour := int(timeRaw / 10000)
	minute := int((timeRaw % 10000) / 100)
	second := int(timeRaw % 100)
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local).Format("2006-01-02 15:04:05")
}
