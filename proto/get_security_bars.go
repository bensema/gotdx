package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type GetSecurityBars struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityBarsRequest
	reply      *GetSecurityBarsReply
}

type GetSecurityBarsRequest struct {
	Market   uint16  // 市场代码。
	Code     [6]byte // 证券代码。
	Category uint16  // K 线周期类别。
	Times    uint16  // 周期倍数。
	Start    uint16  // 起始偏移。
	Count    uint16  // 请求条数。
	Adjust   uint16  // 复权方式。
	Reserved [8]byte // 保留字段。
}

type GetSecurityBarsReply struct {
	Count uint16        // 返回条数。
	List  []SecurityBar // K 线数据列表。
}

type SecurityBar struct {
	Last      float64 // 昨收盘价
	Open      float64 // 开盘价。
	Close     float64 // 收盘价。
	High      float64 // 最高价。
	Low       float64 // 最低价。
	Vol       float64 // 成交量。
	Amount    float64 // 成交额。
	Turnover  float64 // 换手率，按高层接口 best-effort 补齐。
	RisePrice float64 // 涨跌价
	RiseRate  float64 // 涨跌幅
	Year      int     // 年。
	Month     int     // 月。
	Day       int     // 日。
	Hour      int     // 时。
	Minute    int     // 分。
	DateTime  string  // 组合后的时间字符串。
	UpCount   uint16  // 上涨家数，指数类 K 线常见。
	DownCount uint16  // 下跌家数，指数类 K 线常见。
}

func NewGetSecurityBars(req *GetSecurityBarsRequest) *GetSecurityBars {
	// 为了去获取上一个价格的收盘价,从而去计算涨跌价何涨跌幅
	req.Count = req.Count + 1
	obj := new(GetSecurityBars)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityBarsRequest)
	obj.reply = new(GetSecurityBarsReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_SECURITYBARS

	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetSecurityBars) applyRequest(req *GetSecurityBarsRequest) {
	if req.Times == 0 {
		req.Times = 1
	}
	obj.request = req
}

func (obj *GetSecurityBars) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x1c
	obj.reqHeader.PkgLen2 = 0x1c

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetSecurityBars) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	isOffsetReq := obj.respHeader.Method == KMSG_SECURITYBARS_OFFSET

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

		dateTime, ok := decodeDateNum(obj.request.Category, dateNum)
		if !ok {
			return fmt.Errorf("invalid kline datetime: %d", dateNum)
		}

		var openRaw, closeRaw, highRaw, lowRaw int

		openRaw = getprice(data, &pos)
		if isOffsetReq {
			openRaw = lastRaw + openRaw
			closeRaw = getprice(data, &pos) + openRaw
			highRaw = getprice(data, &pos) + openRaw
			lowRaw = getprice(data, &pos) + openRaw
		} else {
			closeRaw = getprice(data, &pos)
			highRaw = getprice(data, &pos)
			lowRaw = getprice(data, &pos)
		}

		vol := getfloat32(data, &pos)
		amount := getfloat32(data, &pos)

		bar := SecurityBar{
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
			DateTime: dateTime.Format("2006-01-02 15:04:05"),
		}
		bar.Last = float64(lastRaw) / 1000.0
		lastRaw = closeRaw

		bar.RiseRate = bar.GetRiseRate()
		bar.RisePrice = bar.GetRisePrice()

		if pos+4 <= len(data) {
			tryDateNum := binary.LittleEndian.Uint32(data[pos : pos+4])
			if _, ok := decodeDateNum(obj.request.Category, tryDateNum); !ok {
				bar.UpCount = binary.LittleEndian.Uint16(data[pos : pos+2])
				bar.DownCount = binary.LittleEndian.Uint16(data[pos+2 : pos+4])
				pos += 4
			}
		}
		//由于在 NewGetSecurityBars 中多请求了一条数据,故而第一条数据我们先舍弃
		if index == 0 {
			continue
		}
		obj.reply.List = append(obj.reply.List, bar)
	}
	obj.reply.Count = obj.reply.Count - 1

	return nil
}

func (bar SecurityBar) GetRisePrice() float64 {
	if bar.Last == 0 {
		//稍微数据准确点，没减去0这么夸张，还是不准的
		return bar.Close - bar.Open
	}
	return bar.Close - bar.Last
}

// RiseRate 涨跌比例/涨跌幅
func (bar SecurityBar) GetRiseRate() float64 {
	if bar.Last == 0 {
		return float64(bar.Close-bar.Open) / float64(bar.Open) * 100
	}
	return float64(bar.Close-bar.Last) / float64(bar.Last) * 100
}

func (obj *GetSecurityBars) Response() *GetSecurityBarsReply {
	return obj.reply
}
