package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type GetIndexBars struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetIndexBarsRequest
	reply      *GetIndexBarsReply

	contentHex string
}

type GetIndexBarsRequest struct {
	Market   uint16
	Code     [6]byte
	Category uint16 // 种类 5分钟  10分钟
	I        uint16 // 未知 填充
	Start    uint16
	Count    uint16
}

type GetIndexBarsReply struct {
	Count uint16
	List  []IndexBar
}

type IndexBar struct {
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

func NewGetIndexBars() *GetIndexBars {
	obj := new(GetIndexBars)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetIndexBarsRequest)
	obj.reply = new(GetIndexBarsReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	//obj.reqHeader.PkgLen1  =
	//obj.reqHeader.PkgLen2  =
	obj.reqHeader.Method = KMSG_INDEXBARS
	obj.contentHex = "00000000000000000000"
	return obj
}

func (obj *GetIndexBars) SetParams(req *GetIndexBarsRequest) {
	obj.request = req
	obj.request.I = 1
}

func (obj *GetIndexBars) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x1c
	obj.reqHeader.PkgLen2 = 0x1c

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	b, err := hex.DecodeString(obj.contentHex)
	buf.Write(b)

	//b, err := hex.DecodeString(obj.contentHex)
	//buf.Write(b)

	//err = binary.Write(buf, binary.LittleEndian, uint16(len(obj.stocks)))

	return buf.Bytes(), err
}

func (obj *GetIndexBars) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	pos += 2

	pre_diff_base := 0
	//lasttime := ""
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := IndexBar{}

		ele.Year, ele.Month, ele.Day, ele.Hour, ele.Minute = getdatetime(int(obj.request.Category), data, &pos)

		//if index == 0 {
		//	ele.Year, ele.Month, ele.Day, ele.Hour, ele.Minute = getdatetime(int(obj.request.Category), data, &pos)
		//} else {
		//	ele.Year, ele.Month, ele.Day, ele.Hour, ele.Minute = getdatetimenow(int(obj.request.Category), lasttime)
		//}
		ele.DateTime = fmt.Sprintf("%d-%02d-%02d %02d:%02d:00", ele.Year, ele.Month, ele.Day, ele.Hour, ele.Minute)

		price_open_diff := getprice(data, &pos)
		price_close_diff := getprice(data, &pos)

		price_high_diff := getprice(data, &pos)
		price_low_diff := getprice(data, &pos)

		var ivol uint32
		binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &ivol)
		ele.Vol = getvolume(int(ivol))
		pos += 4

		var dbvol uint32
		binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &dbvol)
		ele.Amount = getvolume(int(dbvol))
		pos += 4

		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.UpCount)
		pos += 2
		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.DownCount)
		pos += 2

		ele.Open = float64(price_open_diff+pre_diff_base) / 1000.0
		price_open_diff += pre_diff_base

		ele.Close = float64(price_open_diff+price_close_diff) / 1000.0
		ele.High = float64(price_open_diff+price_high_diff) / 1000.0
		ele.Low = float64(price_open_diff+price_low_diff) / 1000.0

		pre_diff_base = price_open_diff + price_close_diff
		//lasttime = ele.DateTime

		obj.reply.List = append(obj.reply.List, ele)
	}
	return err
}

func (obj *GetIndexBars) Reply() *GetIndexBarsReply {
	return obj.reply
}
