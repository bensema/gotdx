package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACTickCharts 表示 0x123E MAC 多日分时协议。
type MACTickCharts struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACTickChartsRequest
	reply      *MACTickChartsReply
}

// MACTickChartsRequest 表示 MAC 多日分时请求。
type MACTickChartsRequest struct {
	Market    uint16
	Code      [22]byte
	QueryDate uint32
	Days      uint16
	One       uint16
	Reserved  [6]byte
}

// MACTickChartsReply 表示 MAC 多日分时响应。
type MACTickChartsReply struct {
	Market       uint16
	Code         string
	Count        uint16
	SendLast     uint8
	PageSize     uint16
	Total        uint16
	Charts       []MACTickChartDay
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

// MACTickChartDay 表示单日多日分时数据。
type MACTickChartDay struct {
	Date     string
	PreClose float64
	Ticks    []MACTickChartItem
}

// MACTickChartItem 表示单个分时点。
type MACTickChartItem struct {
	Time    string
	Price   float64
	Avg     float64
	Vol     uint16
	Unknown uint16
}

// NewMACTickCharts 创建 MAC 多日分时协议对象。
func NewMACTickCharts(req *MACTickChartsRequest) *MACTickCharts {
	obj := &MACTickCharts{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACTickChartsRequest),
		reply:      new(MACTickChartsReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACTICKCHARTS
	obj.request.Days = 5
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACTickCharts) applyRequest(req *MACTickChartsRequest) {
	if req.Days == 0 {
		req.Days = 5
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *MACTickCharts) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_MACTICKCHARTS, payload.Bytes())
}

func (obj *MACTickCharts) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 71 {
		return fmt.Errorf("invalid mac tick charts response length: %d", len(data))
	}

	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:24])

	var dates [5]uint32
	var preCloses [5]float32
	for i := range dates {
		dates[i] = binary.LittleEndian.Uint32(data[24+i*4 : 28+i*4])
		preCloses[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[44+i*4 : 48+i*4]))
	}

	obj.reply.Count = binary.LittleEndian.Uint16(data[64:66])
	obj.reply.SendLast = data[66]
	obj.reply.PageSize = binary.LittleEndian.Uint16(data[67:69])
	obj.reply.Total = binary.LittleEndian.Uint16(data[69:71])

	pos := 71
	type tickWithMinute struct {
		minute uint16
		item   MACTickChartItem
	}

	ticks := make([]tickWithMinute, 0, obj.reply.Total)
	for tickIndex := uint16(0); tickIndex < obj.reply.Total; tickIndex++ {
		if pos+14 > len(data) {
			return fmt.Errorf("invalid mac tick charts item %d", tickIndex)
		}
		minutes := binary.LittleEndian.Uint16(data[pos : pos+2])
		ticks = append(ticks, tickWithMinute{
			minute: minutes,
			item: MACTickChartItem{
				Time:    time.Date(0, 1, 1, int(minutes/60)%24, int(minutes%60), 0, 0, time.Local).Format("15:04:05"),
				Price:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))),
				Avg:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+6 : pos+10]))),
				Vol:     binary.LittleEndian.Uint16(data[pos+10 : pos+12]),
				Unknown: binary.LittleEndian.Uint16(data[pos+12 : pos+14]),
			},
		})
		pos += 14
	}

	if obj.reply.Count == 0 {
		obj.reply.Charts = nil
	} else {
		currentDay := 0
		day := newMACTickChartDay(dates, preCloses, currentDay)
		for i, tick := range ticks {
			day.Ticks = append(day.Ticks, tick.item)
			if currentDay >= int(obj.reply.Count)-1 {
				continue
			}
			if i+1 < len(ticks) && ticks[i+1].minute <= tick.minute {
				obj.reply.Charts = append(obj.reply.Charts, day)
				currentDay++
				day = newMACTickChartDay(dates, preCloses, currentDay)
			}
		}
		obj.reply.Charts = append(obj.reply.Charts, day)
		for len(obj.reply.Charts) < int(obj.reply.Count) {
			obj.reply.Charts = append(obj.reply.Charts, newMACTickChartDay(dates, preCloses, len(obj.reply.Charts)))
		}
	}

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

// Response 返回解析后的 MAC 多日分时响应。
func (obj *MACTickCharts) Response() *MACTickChartsReply {
	return obj.reply
}

func formatMACDate(raw uint32) string {
	year := int(raw / 10000)
	month := int((raw % 10000) / 100)
	day := int(raw % 100)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
}

func newMACTickChartDay(dates [5]uint32, preCloses [5]float32, dayIndex int) MACTickChartDay {
	day := MACTickChartDay{}
	if dayIndex < len(dates) && dates[dayIndex] != 0 {
		day.Date = formatMACDate(dates[dayIndex])
		day.PreClose = float64(preCloses[dayIndex])
	}
	return day
}
