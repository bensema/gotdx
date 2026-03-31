package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type GetUnusual struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetUnusualRequest
	reply      *GetUnusualReply
}

type GetUnusualRequest struct {
	Market uint16
	Start  uint32
	Count  uint32
}

type GetUnusualReply struct {
	Count uint16
	List  []UnusualData
}

type UnusualData struct {
	Index  uint16
	Market uint16
	Code   string
	Time   string
	Desc   string
	Value  string
}

func NewGetUnusual() *GetUnusual {
	obj := new(GetUnusual)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetUnusualRequest)
	obj.reply = new(GetUnusualReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_UNUSUAL
	return obj
}

func (obj *GetUnusual) SetParams(req *GetUnusualRequest) {
	if req.Count == 0 {
		req.Count = 600
	}
	obj.request = req
}

func (obj *GetUnusual) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0c
	obj.reqHeader.PkgLen2 = 0x0c

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetUnusual) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	if err := binary.Read(bytes.NewBuffer(data[:2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}

	for i := uint16(0); i < obj.reply.Count; i++ {
		base := int(i)*32 + 2
		if base+32 > len(data) {
			return io.ErrUnexpectedEOF
		}

		market := binary.LittleEndian.Uint16(data[base : base+2])
		code := Utf8ToGbk(data[base+2 : base+8])
		eventType := data[base+9]
		index := binary.LittleEndian.Uint16(data[base+11 : base+13])
		desc, value := unpackUnusualByType(eventType, data[base+15:base+28])
		hour := int(data[base+29])
		minuteSec := int(binary.LittleEndian.Uint16(data[base+30 : base+32]))

		obj.reply.List = append(obj.reply.List, UnusualData{
			Index:  index,
			Market: market,
			Code:   code,
			Time:   fmt.Sprintf("%02d:%02d:%02d", hour, minuteSec/100, minuteSec%100),
			Desc:   desc,
			Value:  value,
		})
	}

	return nil
}

func (obj *GetUnusual) Reply() *GetUnusualReply {
	return obj.reply
}

func unpackUnusualByType(eventType byte, data []byte) (string, string) {
	if len(data) < 13 {
		return "", ""
	}

	v1 := data[0]
	v2 := math.Float32frombits(binary.LittleEndian.Uint32(data[1:5]))
	v3 := math.Float32frombits(binary.LittleEndian.Uint32(data[5:9]))
	v4 := math.Float32frombits(binary.LittleEndian.Uint32(data[9:13]))

	switch eventType {
	case 0x03:
		if v1 == 0x00 {
			return "主力买入", fmt.Sprintf("%.2f/%.2f", v2, v3)
		}
		return "主力卖出", fmt.Sprintf("%.2f/%.2f", v2, v3)
	case 0x04:
		return "加速拉升", fmt.Sprintf("%.2f%%", v2*100)
	case 0x05:
		return "加速下跌", ""
	case 0x06:
		return "低位反弹", fmt.Sprintf("%.2f%%", v2*100)
	case 0x07:
		return "高位回落", fmt.Sprintf("%.2f%%", v2*100)
	case 0x08:
		return "撑杆跳高", fmt.Sprintf("%.2f%%", v2*100)
	case 0x09:
		return "平台跳水", fmt.Sprintf("%.2f%%", v2*100)
	case 0x0a:
		if v2 < 0 {
			return "单笔冲跌", fmt.Sprintf("%.2f%%", v2*100)
		}
		return "单笔冲涨", fmt.Sprintf("%.2f%%", v2*100)
	case 0x0b:
		if v3 == 0 {
			return "区间放量平", fmt.Sprintf("%.1f倍", v2)
		}
		if v3 < 0 {
			return "区间放量跌", fmt.Sprintf("%.1f倍%.2f%%", v2, v3*100)
		}
		return "区间放量涨", fmt.Sprintf("%.1f倍%.2f%%", v2, v3*100)
	case 0x0c:
		return "区间缩量", ""
	case 0x10:
		return "大单托盘", fmt.Sprintf("%.2f/%.2f", v4, v3)
	case 0x11:
		return "大单压盘", fmt.Sprintf("%.2f/%.2f", v2, v3)
	case 0x12:
		return "大单锁盘", ""
	case 0x13:
		return "竞价试买", fmt.Sprintf("%.2f/%.2f", v2, v3)
	case 0x14:
		if len(data) < 10 {
			return "", ""
		}
		subType := data[1]
		uv2 := math.Float32frombits(binary.LittleEndian.Uint32(data[2:6]))
		uv3 := math.Float32frombits(binary.LittleEndian.Uint32(data[6:10]))
		direction := "涨"
		if v1 != 0x00 {
			direction = "跌"
		}
		desc := ""
		switch subType {
		case 0x01:
			desc = "逼近" + direction + "停"
		case 0x02:
			desc = "封" + direction + "停板"
		case 0x04:
			desc = "封" + direction + "大减"
		case 0x05:
			desc = "打开" + direction + "停"
		}
		return desc, fmt.Sprintf("%.2f/%.2f", uv2, uv3)
	case 0x15:
		desc := "尾盘打压"
		switch v1 {
		case 0x00:
			desc = "尾盘??"
		case 0x01:
			desc = "尾盘对倒"
		case 0x02:
			desc = "尾盘拉升"
		}
		return desc, fmt.Sprintf("%.2f%%/%.2f", v2*100, v3)
	case 0x16:
		if v2 < 0 {
			return "盘中弱势", fmt.Sprintf("%.2f%%", v2*100)
		}
		return "盘中强势", fmt.Sprintf("%.2f%%", v2*100)
	case 0x1d:
		return "急速拉升", fmt.Sprintf("%.2f%%", v2*100)
	case 0x1e:
		return "急速下跌", fmt.Sprintf("%.2f%%", v2*100)
	default:
		return "", ""
	}
}
