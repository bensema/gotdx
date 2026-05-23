package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACAuction 表示 0x123D MAC 竞价协议。
type MACAuction struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACAuctionRequest
	reply      *MACAuctionReply
}

// MACAuctionRequest 表示 MAC 竞价请求。
type MACAuctionRequest struct {
	Market   uint16
	Code     [22]byte
	Start    uint32
	Count    uint32
	Reserved [10]byte
}

// MACAuctionReply 表示 MAC 竞价响应。
type MACAuctionReply struct {
	Market uint16
	Code   string
	Count  uint32
	List   []MACAuctionItem
}

// MACAuctionItem 表示单条 MAC 竞价数据。
type MACAuctionItem struct {
	Time      string
	Price     float64
	Matched   uint32
	Unmatched int32
	Flag      int8
}

// NewMACAuction 创建 MAC 竞价协议对象。
func NewMACAuction(req *MACAuctionRequest) *MACAuction {
	obj := &MACAuction{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACAuctionRequest),
		reply:      new(MACAuctionReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACAUCTION
	obj.request.Count = 500
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACAuction) applyRequest(req *MACAuctionRequest) {
	if req.Count == 0 {
		req.Count = 500
	}
	obj.request = req
}

func (obj *MACAuction) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_MACAUCTION, payload.Bytes())
}

func (obj *MACAuction) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 36 {
		return fmt.Errorf("invalid mac auction response length: %d", len(data))
	}

	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:24])
	obj.reply.Count = binary.LittleEndian.Uint32(data[24:28])

	pos := 36
	for i := uint32(0); i < obj.reply.Count; i++ {
		if pos+16 > len(data) {
			return fmt.Errorf("invalid mac auction item %d", i)
		}
		seconds := binary.LittleEndian.Uint32(data[pos : pos+4])
		unmatched := int32(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))
		matched := binary.LittleEndian.Uint32(data[pos+8 : pos+12])
		if unmatched < 0 {
			unmatched = -unmatched
		}
		flag := int8(1)
		if unmatched < 0 {
			flag = -1
		}
		obj.reply.List = append(obj.reply.List, MACAuctionItem{
			Time:      time.Date(0, 1, 1, int(seconds/3600)%24, int((seconds%3600)/60), int(seconds%60), 0, time.Local).Format("15:04:05"),
			Price:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))),
			Matched:   matched,
			Unmatched: unmatched,
			Flag:      flag,
		})
		pos += 16
	}
	return nil
}

// Response 返回解析后的 MAC 竞价响应。
func (obj *MACAuction) Response() *MACAuctionReply {
	return obj.reply
}
