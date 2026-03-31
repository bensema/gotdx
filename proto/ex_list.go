package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type ExGetCount struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExGetCountReply
}

type ExGetCountReply struct {
	Count uint32
}

func NewExGetCount() *ExGetCount {
	obj := &ExGetCount{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExGetCountReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXCOUNT
	return obj
}

func (obj *ExGetCount) Serialize() ([]byte, error) {
	return serializeExRequest(KMSG_EXCOUNT, nil)
}

func (obj *ExGetCount) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 23 {
		return fmt.Errorf("invalid ex count response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint32(data[19:23])
	return nil
}

func (obj *ExGetCount) Reply() *ExGetCountReply {
	return obj.reply
}

type ExGetCategoryList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExGetCategoryListReply
}

type ExGetCategoryListReply struct {
	Count uint16
	List  []ExCategoryItem
}

type ExCategoryItem struct {
	Market uint8
	Name   string
	Code   uint8
	Abbr   string
}

func NewExGetCategoryList() *ExGetCategoryList {
	obj := &ExGetCategoryList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExGetCategoryListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXCATEGORYLIST
	return obj
}

func (obj *ExGetCategoryList) Serialize() ([]byte, error) {
	return serializeExRequest(KMSG_EXCATEGORYLIST, nil)
}

func (obj *ExGetCategoryList) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 2 {
		return fmt.Errorf("invalid ex category list response length: %d", len(data))
	}

	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	for i := uint16(0); i < obj.reply.Count; i++ {
		base := 2 + int(i)*64
		if base+64 > len(data) {
			return fmt.Errorf("invalid ex category list item %d", i)
		}
		obj.reply.List = append(obj.reply.List, ExCategoryItem{
			Market: data[base],
			Name:   Utf8ToGbk(data[base+1 : base+33]),
			Code:   data[base+33],
			Abbr:   Utf8ToGbk(data[base+34 : base+64]),
		})
	}
	return nil
}

func (obj *ExGetCategoryList) Reply() *ExGetCategoryListReply {
	return obj.reply
}

type ExGetList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetListRequest
	reply      *ExGetListReply
}

type ExGetListRequest struct {
	Start uint32
	Count uint16
}

type ExGetListReply struct {
	Start uint32
	Count uint16
	List  []ExListItem
}

type ExListItem struct {
	Market   uint8
	Category uint8
	Code     string
	Name     string
	Desc     []float64
}

func NewExGetList() *ExGetList {
	obj := &ExGetList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetListRequest),
		reply:      new(ExGetListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXLIST
	return obj
}

func (obj *ExGetList) SetParams(req *ExGetListRequest) {
	obj.request = req
}

func (obj *ExGetList) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXLIST, payload.Bytes())
}

func (obj *ExGetList) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 6 {
		return fmt.Errorf("invalid ex list response length: %d", len(data))
	}

	obj.reply.Start = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Count = binary.LittleEndian.Uint16(data[4:6])
	for i := uint16(0); i < obj.reply.Count; i++ {
		base := 6 + int(i)*64
		if base+64 > len(data) {
			return fmt.Errorf("invalid ex list item %d", i)
		}
		item := ExListItem{
			Market:   data[base],
			Category: data[base+1],
			Code:     Utf8ToGbk(data[base+5 : base+14]),
			Name:     Utf8ToGbk(data[base+14 : base+40]),
			Desc: []float64{
				float64(data[base+2]),
				float64(binary.LittleEndian.Uint16(data[base+3 : base+5])),
				float64(math.Float32frombits(binary.LittleEndian.Uint32(data[base+40 : base+44]))),
				float64(math.Float32frombits(binary.LittleEndian.Uint32(data[base+44 : base+48]))),
				float64(binary.LittleEndian.Uint16(data[base+48 : base+50])),
				float64(binary.LittleEndian.Uint16(data[base+50 : base+52])),
				float64(binary.LittleEndian.Uint16(data[base+52 : base+54])),
				float64(binary.LittleEndian.Uint16(data[base+54 : base+56])),
				float64(binary.LittleEndian.Uint16(data[base+56 : base+58])),
				float64(binary.LittleEndian.Uint16(data[base+58 : base+60])),
				float64(binary.LittleEndian.Uint16(data[base+60 : base+62])),
				float64(binary.LittleEndian.Uint16(data[base+62 : base+64])),
			},
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	return nil
}

func (obj *ExGetList) Reply() *ExGetListReply {
	return obj.reply
}
