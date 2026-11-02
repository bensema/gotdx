package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// GetSecurityList 表示一次分页获取市场内证券代码与基本信息协议的完整流程对象。
//
// 该对象按市场、起始偏移与条数分页拉取证券静态档案。零值不可直接使用，必须通过
// NewGetSecurityList 构造；对象绑定单次查询流程，不支持多协程并发复用。
type GetSecurityList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityListRequest
	reply      *GetSecurityListReply

	contentHex string
}

// GetSecurityListRequest 是 GetSecurityList 分页查询使用的请求参数。
type GetSecurityListRequest struct {
	Market uint16 // 市场代码，`uint16`，取值对应 `types` 包市场枚举。
	Start  uint32 // 起始偏移，`uint32`，从 0 开始按页递增。
	Count  uint32 // 本页请求条数，`uint32`；为 0 时默认每页 1600 只。
	Zero   uint32 // 协议保留占位，填 0。
}

// GetSecurityListReply 是 GetSecurityList 分页查询解析完成后的响应数据。
type GetSecurityListReply struct {
	Count uint16     // 本页实际返回条数，`uint16`。
	List  []Security // 本页证券档案列表，长度与 `Count` 一致。
}

// Security 表示市场内一只证券的静态基本信息，包含代码、名称、昨收价等字段。
type Security struct {
	Code           string  // 6 字节定长证券代码。
	Vol            uint16  // 成交量单位原值，`uint16`，同时回填为 `VolUnit`。
	VolUnit        uint16  // 成交量单位，取值与 `Vol` 同源。
	DecimalPoint   int8    // 价格小数位数，1 字节有符号整数，用于价格缩放参考。
	Name           string  // 16 字节定长证券名称，按 `GBK` 解码。
	PreClose       float64 // 昨收价，单位为元，由 4 字节的单精度浮点数解码。
	Unknown1       float32 // 未确认字段，语义待确认。
	LegacyUnknown1 uint16  // 旧版列表中的首个未知字段，语义待确认。
	Unknown2       uint16  // 尾部未知字段，2 字节`uint16`，语义待确认。
	Unknown3       uint16  // 尾部未知字段，2 字节`uint16`，语义待确认。
}

// NewGetSecurityList 构造证券列表查询对象。
//
// 参数 `req` 指定市场 `Market`、起始偏移 `Start` 与本页条数 `Count`；`Count` 为 0
// 时默认每页 1600 只；`req` 为 `nil` 时使用空请求参数。返回的对象可直接调用
// `BuildRequest` 与 `ParseResponse`。
func NewGetSecurityList(req *GetSecurityListRequest) *GetSecurityList {
	obj := new(GetSecurityList)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityListRequest)
	obj.reply = new(GetSecurityListReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYLIST
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetSecurityList) applyRequest(req *GetSecurityListRequest) {
	// 未指定条数时默认每页 1600 只证券。
	if req.Count == 0 {
		req.Count = 1600
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文。
//
// 返回的字节切片可直接写入连接；`PkgLen1` 与 `PkgLen2` 双写为同一负载长度，
// 缓冲区写入失败时返回错误。
func (obj *GetSecurityList) BuildRequest() ([]byte, error) {
	// 负载 14 字节：市场 2、起始偏移 4、条数 4、保留 4，包头长度按该结构双写。
	obj.reqHeader.PkgLen1 = 2 + 2 + 4 + 4 + 4
	obj.reqHeader.PkgLen2 = 2 + 2 + 4 + 4 + 4

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)

	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的证券列表报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；解码失败时返回错误。
func (obj *GetSecurityList) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	// 头部 2 字节：本页证券条数，不足直接拒绝截断负载。
	rawCount, ok := readRange(data, &pos, 2)
	if !ok {
		return fmt.Errorf("invalid security list response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(rawCount)
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := Security{}
		// 6 字节定长证券代码。
		rawCode, ok := readRange(data, &pos, 6)
		if !ok {
			return fmt.Errorf("invalid security list item %d: code overflow", index)
		}
		ele.Code = string(rawCode)

		// 成交量原值，的无符号短整数；同时作为成交量单位回填。
		rawVol, ok := readRange(data, &pos, 2)
		if !ok {
			return fmt.Errorf("invalid security list item %d: volume overflow", index)
		}
		ele.Vol = binary.LittleEndian.Uint16(rawVol)
		ele.VolUnit = ele.Vol

		// 16 字节定长证券名称，按 `GBK` 解码。
		rawName, ok := readRange(data, &pos, 16)
		if !ok {
			return fmt.Errorf("invalid security list item %d: name overflow", index)
		}
		ele.Name = Utf8ToGbk(rawName)

		// 未知字段：4 字节位模式按单精度浮点数解释，语义待确认。
		rawUnknown1, ok := readRange(data, &pos, 4)
		if !ok {
			return fmt.Errorf("invalid security list item %d: unknown1 overflow", index)
		}
		ele.Unknown1 = math.Float32frombits(binary.LittleEndian.Uint32(rawUnknown1))
		// 价格小数位数，1 字节有符号整数。
		rawDecimal, ok := readRange(data, &pos, 1)
		if !ok {
			return fmt.Errorf("invalid security list item %d: decimal overflow", index)
		}
		ele.DecimalPoint = int8(rawDecimal[0])
		// 昨收价：4 字节的单精度浮点数，由 `getfloat32` 解码并推进 `pos`。
		ele.PreClose = getfloat32(data, &pos)

		// 尾部未知字段 2，2 字节`uint16`，语义待确认。
		rawUnknown2, ok := readRange(data, &pos, 2)
		if !ok {
			return fmt.Errorf("invalid security list item %d: unknown2 overflow", index)
		}
		ele.Unknown2 = binary.LittleEndian.Uint16(rawUnknown2)
		// 尾部未知字段 3，2 字节`uint16`，语义待确认。
		rawUnknown3, ok := readRange(data, &pos, 2)
		if !ok {
			return fmt.Errorf("invalid security list item %d: unknown3 overflow", index)
		}
		ele.Unknown3 = binary.LittleEndian.Uint16(rawUnknown3)

		obj.reply.List = append(obj.reply.List, ele)
	}
	return nil
}

// Response 返回已解析完成的证券列表响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetSecurityList) Response() *GetSecurityListReply {
	return obj.reply
}
