package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type ExGetTickChart struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetTickChartRequest
	reply      *ExGetTickChartReply
}

type ExGetTickChartRequest struct {
	Category uint8
	Code     [23]byte
	Reserved [8]byte
}

type ExGetTickChartReply struct {
	Category uint8
	Code     string
	Count    uint16
	List     []ExTickChartData
}

type ExTickChartData struct {
	Time  string
	Price float64
	Avg   float64
	Vol   int
}

func NewExGetTickChart() *ExGetTickChart {
	obj := &ExGetTickChart{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetTickChartRequest),
		reply:      new(ExGetTickChartReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXTICKCHART
	return obj
}

func (obj *ExGetTickChart) SetParams(req *ExGetTickChartRequest) {
	obj.request = req
}

func (obj *ExGetTickChart) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXTICKCHART, payload.Bytes())
}

func (obj *ExGetTickChart) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 34 {
		return fmt.Errorf("invalid ex tick chart response length: %d", len(data))
	}

	obj.reply.Category = data[0]
	obj.reply.Code = Utf8ToGbk(data[1:32])
	obj.reply.Count = binary.LittleEndian.Uint16(data[32:34])
	pos := 34

	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+18 > len(data) {
			return fmt.Errorf("invalid ex tick chart item %d", i)
		}
		minutes := binary.LittleEndian.Uint16(data[pos : pos+2])
		obj.reply.List = append(obj.reply.List, ExTickChartData{
			Time:  fmt.Sprintf("%02d:%02d", (minutes/60)%24, minutes%60),
			Price: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))),
			Avg:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+6 : pos+10]))),
			Vol:   int(binary.LittleEndian.Uint32(data[pos+10 : pos+14])),
		})
		pos += 18
	}
	return nil
}

func (obj *ExGetTickChart) Reply() *ExGetTickChartReply {
	return obj.reply
}

type ExGetHistoryTickChart struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetHistoryTickChartRequest
	reply      *ExGetHistoryTickChartReply
}

type ExGetHistoryTickChartRequest struct {
	Date     uint32
	Category uint8
	Code     [23]byte
	Reserved [6]byte
	Unknown  uint16
}

type ExGetHistoryTickChartReply struct {
	Category uint8
	Name     string
	Date     string
	AvgPrice float64
	Count    uint16
	List     []ExTickChartData
}

func NewExGetHistoryTickChart() *ExGetHistoryTickChart {
	obj := &ExGetHistoryTickChart{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetHistoryTickChartRequest),
		reply:      new(ExGetHistoryTickChartReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXHISTORYTICKCHART
	return obj
}

func (obj *ExGetHistoryTickChart) SetParams(req *ExGetHistoryTickChartRequest) {
	obj.request = req
}

func (obj *ExGetHistoryTickChart) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXHISTORYTICKCHART, payload.Bytes())
}

func (obj *ExGetHistoryTickChart) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 42 {
		return fmt.Errorf("invalid ex history tick chart response length: %d", len(data))
	}

	obj.reply.Category = data[0]
	obj.reply.Name = Utf8ToGbk(data[1:24])
	dateRaw := binary.LittleEndian.Uint32(data[24:28])
	obj.reply.Date = fmt.Sprintf("%04d-%02d-%02d", dateRaw/10000, (dateRaw%10000)/100, dateRaw%100)
	obj.reply.AvgPrice = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[28:32])))
	obj.reply.Count = binary.LittleEndian.Uint16(data[40:42])
	pos := 42

	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+18 > len(data) {
			return fmt.Errorf("invalid ex history tick chart item %d", i)
		}
		minutes := binary.LittleEndian.Uint16(data[pos : pos+2])
		obj.reply.List = append(obj.reply.List, ExTickChartData{
			Time:  fmt.Sprintf("%02d:%02d", (minutes/60)%24, minutes%60),
			Price: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))),
			Avg:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+6 : pos+10]))),
			Vol:   int(binary.LittleEndian.Uint32(data[pos+10 : pos+14])),
		})
		pos += 18
	}
	return nil
}

func (obj *ExGetHistoryTickChart) Reply() *ExGetHistoryTickChartReply {
	return obj.reply
}

type ExGetChartSampling struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetChartSamplingRequest
	reply      *ExGetChartSamplingReply
}

type ExGetChartSamplingRequest struct {
	Category uint16
	Code     [22]byte
	One      uint16
	Count    uint16
	Reserved [9]byte
}

type ExGetChartSamplingReply struct {
	Category uint16
	Code     string
	Count    uint16
	Prices   []float64
}

func NewExGetChartSampling() *ExGetChartSampling {
	obj := &ExGetChartSampling{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetChartSamplingRequest),
		reply:      new(ExGetChartSamplingReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXCHARTSAMPLING
	obj.request.One = 1
	obj.request.Count = 20
	return obj
}

func (obj *ExGetChartSampling) SetParams(req *ExGetChartSamplingRequest) {
	if req.One == 0 {
		req.One = 1
	}
	if req.Count == 0 {
		req.Count = 20
	}
	obj.request = req
}

func (obj *ExGetChartSampling) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXCHARTSAMPLING, payload.Bytes())
}

func (obj *ExGetChartSampling) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 42 {
		return fmt.Errorf("invalid ex chart sampling response length: %d", len(data))
	}

	obj.reply.Category = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:24])
	obj.reply.Count = binary.LittleEndian.Uint16(data[40:42])
	pos := 42

	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+4 > len(data) {
			return fmt.Errorf("invalid ex chart sampling item %d", i)
		}
		obj.reply.Prices = append(obj.reply.Prices, float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos:pos+4]))))
		pos += 4
	}
	return nil
}

func (obj *ExGetChartSampling) Reply() *ExGetChartSamplingReply {
	return obj.reply
}
