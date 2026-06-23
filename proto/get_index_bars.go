package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

type GetIndexBars struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetIndexBarsRequest
	reply      *GetIndexBarsReply
}

type GetIndexBarsRequest struct {
	Market   uint16
	Code     [6]byte
	Category uint16
	Times    uint16
	Start    uint16
	Count    uint16
	Adjust   uint16
	Reserved [8]byte
}

type GetIndexBarsReply struct {
	Count uint16
	List  []IndexBar
}

type IndexBar struct {
	Last      float64
	RisePrice float64 // 涨跌价
	RiseRate  float64 // 涨跌幅
	Open      float64
	Close     float64
	High      float64
	Low       float64
	Vol       float64
	Amount    float64
	Year      int
	Month     int
	Day       int
	Hour      int
	Minute    int
	DateTime  time.Time
	UpCount   uint16
	DownCount uint16
}

func NewGetIndexBars(req *GetIndexBarsRequest) *GetIndexBars {

	obj := new(GetIndexBars)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetIndexBarsRequest)
	obj.reply = new(GetIndexBarsReply)
	req.Count = req.Count + 1

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_INDEXBARS
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetIndexBars) applyRequest(req *GetIndexBarsRequest) {
	if req.Times == 0 {
		req.Times = 1
	}
	obj.request = req
}

func (obj *GetIndexBars) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x1c
	obj.reqHeader.PkgLen2 = 0x1c

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetIndexBars) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2
	var lastRaw int // 昨收盘价
	for index := uint16(0); index < obj.reply.Count; index++ {
		var dateNum uint32
		if err := binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &dateNum); err != nil {
			return err
		}
		pos += 4

		dateTime, ok := decodeDateNum(obj.request.Category, dateNum, true)
		if !ok {
			return fmt.Errorf("invalid kline datetime: %d", dateNum)
		}
		// openRaw = lastRaw + openRaw

		openRaw := getprice(data, &pos)
		openRaw = lastRaw + openRaw
		closeRaw := getprice(data, &pos)
		highRaw := getprice(data, &pos)
		lowRaw := getprice(data, &pos)
		vol := getfloat32(data, &pos)
		amount := getfloat32(data, &pos)

		bar := IndexBar{
			Open:     float64(openRaw) / 1000.0,
			Close:    float64(closeRaw) / 1000.0,
			High:     float64(highRaw) / 1000.0,
			Low:      float64(lowRaw) / 1000.0,
			Vol:      vol,
			Amount:   amount,
			Year:     dateTime.Year(),
			Month:    int(dateTime.Month()),
			Day:      dateTime.Day(),
			Hour:     dateTime.Hour(),
			Minute:   dateTime.Minute(),
			DateTime: dateTime,
		}
		bar.Last = float64(lastRaw) / 1000.0
		bar.RiseRate = bar.GetRiseRate()
		bar.RisePrice = bar.GetRisePrice()
		lastRaw = closeRaw

		if pos+4 <= len(data) {
			tryDateNum := binary.LittleEndian.Uint32(data[pos : pos+4])
			if _, ok := decodeDateNum(obj.request.Category, tryDateNum, true); !ok {
				bar.UpCount = binary.LittleEndian.Uint16(data[pos : pos+2])
				bar.DownCount = binary.LittleEndian.Uint16(data[pos+2 : pos+4])
				pos += 4
			}
		}
		if index == 0 {
			continue
		}

		obj.reply.List = append(obj.reply.List, bar)
	}
	obj.reply.Count = obj.reply.Count - 1

	return nil
}

func (bar IndexBar) GetRisePrice() float64 {
	if bar.Last == 0 {
		//稍微数据准确点，没减去0这么夸张，还是不准的
		return bar.Close - bar.Open
	}
	return bar.Close - bar.Last
}

// RiseRate 涨跌比例/涨跌幅
func (bar IndexBar) GetRiseRate() float64 {
	if bar.Last == 0 {
		return float64(bar.Close-bar.Open) / float64(bar.Open) * 100
	}
	return float64(bar.Close-bar.Last) / float64(bar.Last) * 100
}

func (obj *GetIndexBars) Response() *GetIndexBarsReply {
	return obj.reply
}
