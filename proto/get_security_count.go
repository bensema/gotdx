package proto

import (
	"bytes"
	"encoding/binary"
)

// GetSecurityCount 表示一次查询指定市场内证券数量协议的完整流程对象。
//
// 该协议通常在分页拉取证券列表之前调用，用于获取市场证券总数以计算分页范围。
// 零值不可直接使用，必须通过 NewGetSecurityCount 构造；对象绑定单次查询流程，
// 不支持多协程并发复用。
type GetSecurityCount struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityCountRequest
	reply      *GetSecurityCountReply
	contentHex string
}

// GetSecurityCountRequest 是 GetSecurityCount 查询使用的请求参数。
type GetSecurityCountRequest struct {
	Market uint16 // 市场代码。
	Date   uint32 // 交易日期；为 0 时默认取当日。
}

// GetSecurityCountReply 是 GetSecurityCount 查询解析完成后的响应数据。
type GetSecurityCountReply struct {
	Count uint16 // 市场内证券总数。
}

// NewGetSecurityCount 构造证券数量查询对象；`req` 为 `nil` 时使用空请求参数，返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetSecurityCount(req *GetSecurityCountRequest) *GetSecurityCount {
	obj := new(GetSecurityCount)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityCountRequest)
	obj.reply = new(GetSecurityCountReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYCOUNT
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetSecurityCount) applyRequest(req *GetSecurityCountRequest) {
	// 日期未填时默认取当日，格式为年月日拼接的整数。
	if req.Date == 0 {
		req.Date = todayDate()
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文；缓冲区写入失败时返回错误。
func (obj *GetSecurityCount) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2 + 2 + 4
	obj.reqHeader.PkgLen2 = 2 + 2 + 4

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的证券数量报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；报文长度不足时返回错误。
func (obj *GetSecurityCount) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 响应体仅含前 2 字节：的无符号短整数表示市场证券总数。
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	return nil
}

// Response 返回已解析完成的证券数量响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetSecurityCount) Response() *GetSecurityCountReply {
	return obj.reply
}
