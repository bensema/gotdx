package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACSymbolInfo 查询单只标的的股票摘要行情，包含身份、日期时间、开高低收、量额与内外盘等字段。
//
// 使用时先由 NewMACSymbolInfo 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACSymbolInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACSymbolInfoRequest
	reply      *MACSymbolInfoReply
}

// MACSymbolInfoRequest 保存股票摘要查询的市场与标的代码参数，其余字段为协议保留占位。
type MACSymbolInfoRequest struct {
	Market   uint16   // 市场代码，两字节，取值见 `types.Market*` 常量。
	Code     [22]byte // 证券代码，GBK 编码定长 22 字节，不足补空字符。
	One      uint32   // 协议固定标志位，固定填 1，为 0 时由构造函数回填 1，语义待确认。
	Reserved [12]byte // 协议保留填充字节，填零。
}

// MACSymbolInfoReply 保存股票摘要响应的全部字段；其中三个保留槽位的原始语义待确认，
// 仅按线上类型原样解码保存，未据此推断额外含义。
type MACSymbolInfoReply struct {
	Market        uint16    // 市场代码，两字节。
	Code          string    // 证券代码，GBK 解码。
	Name          string    // 证券名称，GBK 解码。
	DateTime      time.Time // 行情对应的本地时间。
	Activity      uint32    // 活跃度指标，四字节，口径语义待确认。
	PreClose      float64   // 昨收价，单精度浮点还原。
	Open          float64   // 开盘价，单精度浮点还原。
	High          float64   // 最高价，单精度浮点还原。
	Low           float64   // 最低价，单精度浮点还原。
	Close         float64   // 收盘价，单精度浮点还原。
	Momentum      float64   // 动量指标，单精度浮点还原，语义待确认。
	Vol           uint32    // 成交量，四字节。
	Amount        float64   // 成交额，单精度浮点还原。
	InsideVolume  uint32    // 内盘成交量，四字节。
	OutsideVolume uint32    // 外盘成交量，四字节。
	Decimal       uint16    // 价格小数位数，两字节。
	UnknownA      uint32    // 协议保留槽位（150:154），四字节，语义待确认。
	UnknownB      float64   // 协议保留槽位（154:158），单精度浮点还原，语义待确认。
	UnknownC      uint32    // 协议保留槽位（178:182），四字节，语义待确认。
	VR            float64   // 量比，单精度浮点还原。
	Turnover      float64   // 换手率，单精度浮点还原。
	Avg           float64   // 均价，单精度浮点还原。
}

// NewMACSymbolInfo 创建股票摘要查询对象；请求参数为空时使用内置默认参数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`Market` 为市场代码、`Code` 为证券代码；
// `One` 为 0 时回填默认值 1。
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

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACSYMBOLINFO` 与请求头封装成完整报文。
func (obj *MACSymbolInfo) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACSYMBOLINFO, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足 194 字节时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACSymbolInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 响应为 194 字节定长布局，长度不足即判定为非法报文，避免后续切片越界。
	if len(data) < 194 {
		return fmt.Errorf("invalid mac symbol info response length: %d", len(data))
	}

	// 身份段：市场（8:10）、GBK 代码（10:32）、GBK 名称（32:76）；76 到 96 为保留间隙。
	obj.reply.Market = binary.LittleEndian.Uint16(data[8:10])
	obj.reply.Code = Utf8ToGbk(data[10:32])
	obj.reply.Name = Utf8ToGbk(data[32:76])
	// 日期（96:100）与时间（100:104）为打包整数，组合为本地时间；104 到 108 为活跃度字段。
	dateRaw := binary.LittleEndian.Uint32(data[96:100])
	timeRaw := binary.LittleEndian.Uint32(data[100:104])
	obj.reply.DateTime = formatMACQuoteDateTime(dateRaw, timeRaw)
	obj.reply.Activity = binary.LittleEndian.Uint32(data[104:108])
	// 偏移 108 到 132 依次为昨收、开、高、低、收、动量，均为单精度浮点。
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[108:112])))
	obj.reply.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[112:116])))
	obj.reply.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[116:120])))
	obj.reply.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[120:124])))
	obj.reply.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[124:128])))
	obj.reply.Momentum = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[128:132])))
	// 132:148 为成交量、成交额、内盘量、外盘量。
	obj.reply.Vol = binary.LittleEndian.Uint32(data[132:136])
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[136:140])))
	obj.reply.InsideVolume = binary.LittleEndian.Uint32(data[140:144])
	obj.reply.OutsideVolume = binary.LittleEndian.Uint32(data[144:148])
	obj.reply.Decimal = binary.LittleEndian.Uint16(data[148:150])
	// 150:158 两个保留槽位（语义待确认）；158:178 为未映射间隙。
	obj.reply.UnknownA = binary.LittleEndian.Uint32(data[150:154])
	obj.reply.UnknownB = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[154:158])))
	obj.reply.UnknownC = binary.LittleEndian.Uint32(data[178:182])
	// 尾部 182:194：量比、换手率、均价。
	obj.reply.VR = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[182:186])))
	obj.reply.Turnover = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[186:190])))
	obj.reply.Avg = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[190:194])))
	return nil
}

// Response 返回解码完成的股票摘要响应；未调用 ParseResponse 前内容为零值。
func (obj *MACSymbolInfo) Response() *MACSymbolInfoReply {
	return obj.reply
}
