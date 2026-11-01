package proto

import (
	"bytes"
	"encoding/binary"
)

// GetIndexMomentum 表示一次指数动量序列查询协议的完整流程对象。
//
// 该对象持有请求头、请求参数与响应数据；零值不可直接使用，必须通过
// NewGetIndexMomentum 构造。对象绑定单次查询流程，不支持多协程并发复用。
type GetIndexMomentum struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetIndexMomentumRequest
	reply      *GetIndexMomentumReply
}

// GetIndexMomentumRequest 是 GetIndexMomentum 查询使用的请求参数。
type GetIndexMomentumRequest struct {
	Market uint16  // 市场代码。
	Code   [6]byte // 指数代码。
}

// GetIndexMomentumReply 是 GetIndexMomentum 查询解析完成后的响应数据。
type GetIndexMomentumReply struct {
	Count  uint16 // 返回条数。
	Values []int  // 累计动量值序列。
}

// NewGetIndexMomentum 构造指数动量查询对象；`req` 为 `nil` 时使用空请求参数，返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetIndexMomentum(req *GetIndexMomentumRequest) *GetIndexMomentum {
	obj := new(GetIndexMomentum)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetIndexMomentumRequest)
	obj.reply = new(GetIndexMomentumReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_INDEXMOMENTUM
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetIndexMomentum) applyRequest(req *GetIndexMomentumRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文；缓冲区写入失败时返回错误。
func (obj *GetIndexMomentum) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0a
	obj.reqHeader.PkgLen2 = 0x0a

	// 负载固定 10 字节：市场 `uint16`、6 字节代码、2 字节保留字段。
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的指数动量报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；报文长度不足或解码失败
// 时返回错误。
func (obj *GetIndexMomentum) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	// 头部 2 字节：动量条数。
	if err := binary.Read(bytes.NewBuffer(data[:2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2

	startMomentum := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		// `getprice` 按类 `UTF-8` 变长编码读取带符号增量，并自动推进 `pos`。
		momentum := getprice(data, &pos)
		// 服务端下发的是相邻增量差值，本地逐条累加得到累计动量。
		startMomentum += momentum
		obj.reply.Values = append(obj.reply.Values, startMomentum)
	}

	return nil
}

// Response 返回已解析完成的指数动量响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetIndexMomentum) Response() *GetIndexMomentumReply {
	return obj.reply
}
