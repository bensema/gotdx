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
	Market   uint16
	Code     [6]byte
	Category uint16
	Times    uint16
	Start    uint16
	Count    uint16
	Adjust   uint16
	Reserved [8]byte
}

type GetSecurityBarsReply struct {
	Count uint16
	List  []SecurityBar
}

type SecurityBar struct {
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
	DateTime  string
	UpCount   uint16
	DownCount uint16
}

func NewGetSecurityBars() *GetSecurityBars {
	obj := new(GetSecurityBars)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityBarsRequest)
	obj.reply = new(GetSecurityBarsReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_SECURITYBARS
	return obj
}

func (obj *GetSecurityBars) SetParams(req *GetSecurityBarsRequest) {
	if req.Times == 0 {
		req.Times = 1
	}
	obj.request = req
}

func (obj *GetSecurityBars) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x1c
	obj.reqHeader.PkgLen2 = 0x1c

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetSecurityBars) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2

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

		openRaw := getprice(data, &pos)
		closeRaw := getprice(data, &pos)
		highRaw := getprice(data, &pos)
		lowRaw := getprice(data, &pos)
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

		if pos+4 <= len(data) {
			tryDateNum := binary.LittleEndian.Uint32(data[pos : pos+4])
			if _, ok := decodeDateNum(obj.request.Category, tryDateNum); !ok {
				bar.UpCount = binary.LittleEndian.Uint16(data[pos : pos+2])
				bar.DownCount = binary.LittleEndian.Uint16(data[pos+2 : pos+4])
				pos += 4
			}
		}

		obj.reply.List = append(obj.reply.List, bar)
	}

	return nil
}

func (obj *GetSecurityBars) Reply() *GetSecurityBarsReply {
	return obj.reply
}
