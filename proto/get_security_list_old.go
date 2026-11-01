package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// GetSecurityListOld 表示一次旧版证券列表协议的完整流程对象，返回结构与新版一致。
//
// 与新版 GetSecurityList 的差异集中在响应报文：旧版每条记录固定 29 字节，字段按
// 绝对偏移读取。零值不可直接使用，必须通过 NewGetSecurityListOld 构造；对象绑定
// 单次查询流程，不支持多协程并发复用。
type GetSecurityListOld struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityListOldRequest
	reply      *GetSecurityListOldReply
}

// GetSecurityListOldRequest 是 GetSecurityListOld 查询使用的请求参数。
type GetSecurityListOldRequest struct {
	Market uint16 // 市场代码，`uint16`，取值对应 `types` 包市场枚举。
	Start  uint16 // 起始偏移，`uint16`，从 0 开始按页递增。
}

// GetSecurityListOldReply 是 GetSecurityListOld 查询解析完成后的响应数据。
type GetSecurityListOldReply struct {
	Count uint16     // 本页实际返回条数，`uint16`。
	List  []Security // 旧版协议解析出的证券列表，复用新版 `Security` 结构。
}

// NewGetSecurityListOld 构造旧版证券列表查询对象。
//
// 参数 `req` 指定市场 `Market` 与起始偏移 `Start`；`req` 为 `nil` 时使用空请求参数。
// 返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetSecurityListOld(req *GetSecurityListOldRequest) *GetSecurityListOld {
	obj := &GetSecurityListOld{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(GetSecurityListOldRequest),
		reply:      new(GetSecurityListOldReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYLIST_OLD
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetSecurityListOld) applyRequest(req *GetSecurityListOldRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文。
//
// 返回的字节切片可直接写入连接；`PkgLen1` 与 `PkgLen2` 双写为同一负载长度，
// 缓冲区写入失败时返回错误。
func (obj *GetSecurityListOld) BuildRequest() ([]byte, error) {
	// 负载 6 字节：市场 `uint16` 占 2、起始偏移 `uint16` 占 2，另含 2 字节协议保留。
	obj.reqHeader.PkgLen1 = 6
	obj.reqHeader.PkgLen2 = 6
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	err := binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 按旧版协议格式解析证券列表报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；报文长度不足或记录越界
// 时返回错误。
func (obj *GetSecurityListOld) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 至少需要容纳 2 字节条数，否则视为非法报文。
	if len(data) < 2 {
		return fmt.Errorf("invalid security list old response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 单条记录固定 29 字节，越界即终止解析并报错。
		if pos+29 > len(data) {
			return fmt.Errorf("invalid security list old item %d", i)
		}
		// 旧版协议在偏移 16 处的无符号短整数同时回填两个未知字段。
		legacyUnknown1 := binary.LittleEndian.Uint16(data[pos+16 : pos+18])
		item := Security{
			Code:           Utf8ToGbk(data[pos : pos+6]),
			Vol:            binary.LittleEndian.Uint16(data[pos+6 : pos+8]),
			Name:           Utf8ToGbk(data[pos+8 : pos+16]),
			Unknown1:       float32(legacyUnknown1),
			LegacyUnknown1: legacyUnknown1,
			Unknown2:       binary.LittleEndian.Uint16(data[pos+25 : pos+27]),
			Unknown3:       binary.LittleEndian.Uint16(data[pos+27 : pos+29]),
		}
		item.VolUnit = item.Vol
		// 偏移 20 为价格小数位数，1 字节有符号整数。
		item.DecimalPoint = int8(data[pos+20])
		// 偏移 21 起 4 字节的单精度浮点数昨收价。
		item.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+21 : pos+25])))
		obj.reply.List = append(obj.reply.List, item)
		// 整条记录定长 29 字节，顺序推进到下一条。
		pos += 29
	}
	return nil
}

// Response 返回已解析完成的旧版证券列表响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetSecurityListOld) Response() *GetSecurityListOldReply {
	return obj.reply
}
