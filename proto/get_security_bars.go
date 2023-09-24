package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type GetSecurityBars struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityBarsRequest
	reply      *GetSecurityBarsReply

	contentHex string
}

type GetSecurityBarsRequest struct {
	Market   uint16
	Code     [6]byte
	Category uint16 // 种类 5分钟  10分钟
	I        uint16 // 未知 填充
	Start    uint16
	Count    uint16
}

type GetSecurityBarsReply struct {
	Count uint16
	List  []SecurityBar
}

type SecurityBar struct {
	Open     float64
	Close    float64
	High     float64
	Low      float64
	Vol      float64
	Amount   float64
	Year     int
	Month    int
	Day      int
	Hour     int
	Minute   int
	DateTime string
	//UpCount   uint16
	//DownCount uint16
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
	//obj.reqHeader.PkgLen1  =
	//obj.reqHeader.PkgLen2  =
	obj.reqHeader.Method = KMSG_SECURITYBARS
	obj.contentHex = "00000000000000000000"
	return obj
}

func (obj *GetSecurityBars) SetParams(req *GetSecurityBarsRequest) {
	obj.request = req
	obj.request.I = 1
}

func (obj *GetSecurityBars) Serialize() ([]byte, error) {
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

// 结果数据都是\n,\t分隔的中文字符串，比如查询K线数据，返回的结果字符串就形如
// /“时间\t开盘价\t收盘价\t最高价\t最低价\t成交量\t成交额\n
// /20150519\t4.644000\t4.732000\t4.747000\t4.576000\t146667487\t683638848.000000\n
// /20150520\t4.756000\t4.850000\t4.960000\t4.756000\t353161092\t1722953216.000000”
func (obj *GetSecurityBars) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	pos += 2

	pre_diff_base := 0

	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := SecurityBar{}
		ele.Year, ele.Month, ele.Day, ele.Hour, ele.Minute = getdatetime(int(obj.request.Category), data, &pos)

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

		ele.Open = float64(price_open_diff+pre_diff_base) / 1000.0
		price_open_diff += pre_diff_base

		ele.Close = float64(price_open_diff+price_close_diff) / 1000.0
		ele.High = float64(price_open_diff+price_high_diff) / 1000.0
		ele.Low = float64(price_open_diff+price_low_diff) / 1000.0

		pre_diff_base = price_open_diff + price_close_diff

		obj.reply.List = append(obj.reply.List, ele)
	}
	return err
}

func (obj *GetSecurityBars) Reply() *GetSecurityBarsReply {
	return obj.reply
}
