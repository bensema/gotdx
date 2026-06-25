package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACSymbolInfo 表示 0x122A MAC 股票摘要协议。
type MACSymbolInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACSymbolInfoRequest
	reply      *MACSymbolInfoReply
}

// MACSymbolInfoRequest 表示 MAC 股票摘要请求。
type MACSymbolInfoRequest struct {
	Market   uint16
	Code     [22]byte
	One      uint32
	Reserved [12]byte
}

// MACSymbolInfoReply 表示 MAC 股票摘要响应。
type MACSymbolInfoReply struct {
	Market        uint16
	Code          string
	Name          string
	DateTime      time.Time
	Activity      uint32
	PreClose      float64
	Open          float64
	High          float64
	Low           float64
	Close         float64
	Momentum      float64
	Vol           uint32
	Amount        float64
	InsideVolume  uint32
	OutsideVolume uint32
	Decimal       uint16
	UnknownA      uint32
	UnknownB      float64
	UnknownC      uint32
	VR            float64
	Turnover      float64
	Avg           float64
}

// NewMACSymbolInfo 创建 MAC 股票摘要协议对象。
func NewMACSymbolInfo(req *MACSymbolInfoRequest) *MACSymbolInfo {
	obj := &MACSymbolInfo{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACSymbolInfoRequest),
		reply:      new(MACSymbolInfoReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACSYMBOLINFO
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACSymbolInfo) applyRequest(req *MACSymbolInfoRequest) {
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *MACSymbolInfo) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_MACSYMBOLINFO, payload.Bytes())
}

func (obj *MACSymbolInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 194 {
		return fmt.Errorf("invalid mac symbol info response length: %d", len(data))
	}

	obj.reply.Market = binary.LittleEndian.Uint16(data[8:10])
	obj.reply.Code = Utf8ToGbk(data[10:32])
	obj.reply.Name = Utf8ToGbk(data[32:76])
	dateRaw := binary.LittleEndian.Uint32(data[96:100])
	timeRaw := binary.LittleEndian.Uint32(data[100:104])
	obj.reply.DateTime = formatMACQuoteDateTime(dateRaw, timeRaw)
	obj.reply.Activity = binary.LittleEndian.Uint32(data[104:108])
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[108:112])))
	obj.reply.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[112:116])))
	obj.reply.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[116:120])))
	obj.reply.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[120:124])))
	obj.reply.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[124:128])))
	obj.reply.Momentum = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[128:132])))
	obj.reply.Vol = binary.LittleEndian.Uint32(data[132:136])
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[136:140])))
	obj.reply.InsideVolume = binary.LittleEndian.Uint32(data[140:144])
	obj.reply.OutsideVolume = binary.LittleEndian.Uint32(data[144:148])
	obj.reply.Decimal = binary.LittleEndian.Uint16(data[148:150])
	obj.reply.UnknownA = binary.LittleEndian.Uint32(data[150:154])
	obj.reply.UnknownB = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[154:158])))
	obj.reply.UnknownC = binary.LittleEndian.Uint32(data[178:182])
	obj.reply.VR = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[182:186])))
	obj.reply.Turnover = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[186:190])))
	obj.reply.Avg = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[190:194])))
	return nil
}

// Response 返回解析后的 MAC 股票摘要响应。
func (obj *MACSymbolInfo) Response() *MACSymbolInfoReply {
	return obj.reply
}
