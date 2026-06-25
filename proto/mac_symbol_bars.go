package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

type MACSymbolBars struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACSymbolBarsRequest
	reply      *MACSymbolBarsReply
}

type MACSymbolBarsRequest struct {
	Market   uint16
	Code     [22]byte
	Period   uint16
	Times    uint16
	Start    uint32
	Count    uint16
	Adjust   uint16
	Flag1    int8
	Flag2    int8
	Flag3    int8
	Flag4    int8
	Zero     uint16
	Reserved [4]byte
}

type MACSymbolBarsReply struct {
	Market       uint16
	Code         string
	Period       uint8
	Unknown      uint16
	Count        uint16
	Start        uint32
	List         []MACSymbolBar
	Name         string
	Decimal      uint8
	Category     uint16
	VolUnit      float64
	DateTime     time.Time
	PreClose     float64
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Momentum     float64
	Vol          uint32
	Amount       float64
	Turnover     float64
	Avg          float64
	Industry     uint32
	IndustryCode string
}

type MACSymbolBar struct {
	DateTime    time.Time
	Open        float64
	High        float64
	Low         float64
	Close       float64
	Amount      float64
	Vol         float64
	FloatShares float64
	Turnover    float64
	Last        float64
	RisePrice   float64 // 涨跌价
	RiseRate    float64 // 涨跌幅
}

func NewMACSymbolBars(req *MACSymbolBarsRequest) *MACSymbolBars {
	obj := &MACSymbolBars{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACSymbolBarsRequest),
		reply:      new(MACSymbolBarsReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACSYMBOLBARS
	obj.request.Times = 1
	obj.request.Flag1 = 1
	obj.request.Flag2 = 1
	obj.request.Flag4 = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACSymbolBars) applyRequest(req *MACSymbolBarsRequest) {
	if req.Times == 0 {
		req.Times = 1
	}
	if req.Flag1 == 0 {
		req.Flag1 = 1
	}
	if req.Flag2 == 0 {
		req.Flag2 = 1
	}
	if req.Flag4 == 0 {
		req.Flag4 = 1
	}
	req.Count = req.Count + 1
	obj.request = req
}

func (obj *MACSymbolBars) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_MACSYMBOLBARS, payload.Bytes())
}

func (obj *MACSymbolBars) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 33 {
		return fmt.Errorf("invalid mac symbol bars response length: %d", len(data))
	}

	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:14])
	obj.reply.Period = data[24]
	obj.reply.Unknown = binary.LittleEndian.Uint16(data[25:27])
	obj.reply.Count = binary.LittleEndian.Uint16(data[27:29])
	obj.reply.Start = binary.LittleEndian.Uint32(data[29:33])

	formatTDXTime := obj.reply.Period < 4 || obj.reply.Period == 7 || obj.reply.Period == 8
	pos := 33
	var lastRaw float64 // 昨收盘价
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+36 > len(data) {
			return fmt.Errorf("invalid mac symbol bar item %d", i)
		}
		ymd := binary.LittleEndian.Uint32(data[pos : pos+4])
		seconds := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		item := MACSymbolBar{
			DateTime:    combineMACDateTime(ymd, seconds, formatTDXTime),
			Open:        float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))),
			High:        float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))),
			Low:         float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+16 : pos+20]))),
			Close:       float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+20 : pos+24]))),
			Amount:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+24 : pos+28]))),
			Vol:         float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+28 : pos+32]))),
			FloatShares: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+32 : pos+36]))),
		}
		item.Last = lastRaw
		lastRaw = item.Close
		item.RiseRate = item.GetRiseRate()
		item.RisePrice = item.GetRisePrice()
		pos += 36
		if i == 0 {
			continue
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	obj.reply.Count = obj.reply.Count - 1

	if pos+120 > len(data) {
		return nil
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
	obj.reply.IndustryCode = macIndustryBoardSymbol(obj.reply.Industry)

	return nil
}

func (bar MACSymbolBar) GetRisePrice() float64 {
	if bar.Last == 0 {
		//稍微数据准确点，没减去0这么夸张，还是不准的
		return bar.Close - bar.Open
	}
	return bar.Close - bar.Last
}

// RiseRate 涨跌比例/涨跌幅
func (bar MACSymbolBar) GetRiseRate() float64 {
	if bar.Last == 0 {
		return (bar.Close - bar.Open) / (bar.Open) * 100
	}
	return (bar.Close - bar.Last) / (bar.Last) * 100
}

func (obj *MACSymbolBars) Response() *MACSymbolBarsReply {
	return obj.reply
}

func combineMACDateTime(ymd uint32, seconds uint32, formatTDXTime bool) time.Time {
	year := int(ymd / 10000)
	month := int((ymd % 10000) / 100)
	day := int(ymd % 100)
	hours := int(seconds / 3600)
	minutes := int((seconds % 3600) / 60)

	ts := time.Date(year, time.Month(month), day, hours, minutes, 0, 0, time.Local)
	if formatTDXTime && ts.Hour() <= 5 {
		return ts.Add(24 * time.Hour)
	}
	return ts
}
