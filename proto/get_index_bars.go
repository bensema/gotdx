package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"github.com/spf13/cast"
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
	PreClose  float64 // 昨收价
	LastClose float64 // 前收价
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
	if len(data) < 2 {
		return errors.New("数据长度不足")
	}
	obj.respHeader = header
	obj.reply.Count = Uint16(data[:2])
	data = data[2:]
	newCategory := cast.ToInt(obj.request.Category)
	var last int
	for index := uint16(0); index < obj.reply.Count; index++ {
		dateTime := GetTime([4]byte(data[:4]), obj.request.Category)
		bar := IndexBar{
			DateTime: dateTime,
			Year:     dateTime.Year(),
			Month:    int(dateTime.Month()),
			Day:      dateTime.Day(),
			Hour:     dateTime.Hour(),
			Minute:   dateTime.Minute(),
		}
		var open int
		data, open = GetPrice(data[4:])
		var close int
		data, close = GetPrice(data)
		var high int
		data, high = GetPrice(data)
		var low int
		data, low = GetPrice(data)

		bar.PreClose = float64(last) / 1000.0
		bar.LastClose = float64(last) / 1000.0
		bar.Open = float64(open) / 1000.0
		bar.Close = float64(close) / 1000.0
		bar.High = float64(high) / 1000.0
		bar.Low = float64(low) / 1000.0
		last = close

		bar.Vol = getVolume(Uint32(data[:4]))
		data = data[4:]

		switch newCategory {
		case KLINE_TYPE_EXHQ_1MIN, KLINE_TYPE_1MIN, KLINE_TYPE_5MIN, KLINE_TYPE_15MIN, KLINE_TYPE_30MIN, KLINE_TYPE_1HOUR:
			bar.Vol /= 100
		}
		bar.Amount = getVolume(Uint32(data[:4]))
		data = data[4:]

		bar.Vol *= 100
		bar.UpCount = cast.ToUint16(Int([]byte{data[1], data[0]}))
		bar.DownCount = cast.ToUint16(Int([]byte{data[3], data[2]}))
		data = data[4:]

		bar.RiseRate = bar.GetRiseRate()
		bar.RisePrice = bar.GetRisePrice()

		if index == 0 {
			continue
		}
		obj.reply.List = append(obj.reply.List, bar)
	}
	obj.reply.List = FixKlineTimeIndexBar(obj.reply.List)
	obj.reply.Count = obj.reply.Count - 1

	return nil
}

func (bar IndexBar) GetRisePrice() float64 {
	if bar.PreClose == 0 {
		//稍微数据准确点，没减去0这么夸张，还是不准的
		return bar.Close - bar.Open
	}
	return bar.Close - bar.PreClose
}

// RiseRate 涨跌比例/涨跌幅
func (bar IndexBar) GetRiseRate() float64 {
	if bar.PreClose == 0 {
		return float64(bar.Close-bar.Open) / float64(bar.Open) * 100
	}
	return float64(bar.Close-bar.PreClose) / float64(bar.PreClose) * 100
}

func (obj *GetIndexBars) Response() *GetIndexBarsReply {
	return obj.reply
}

func FixKlineTimeIndexBar(ks []IndexBar) []IndexBar {
	if len(ks) == 0 {
		return ks
	}
	now := time.Now()
	//只有当天下午13~15点之间才会出现的时间问题
	node1 := time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, now.Location())
	node2 := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())
	if ks[len(ks)-1].DateTime.Unix() < node1.Unix() || ks[len(ks)-1].DateTime.Unix() > node2.Unix() {
		return ks
	}
	ls := ks
	if len(ls) >= 120 {
		ls = ls[len(ls)-120:]
	}
	for i, v := range ls {
		if v.DateTime.Unix() == node1.Unix() {
			ls[i].DateTime = time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, now.Location())
		}
	}
	return ks
}
