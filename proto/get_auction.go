package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// GetAuction 封装集合竞价（盘中竞价明细）查询的完整请求与响应处理流程。
//
// 典型用法：构造对象后编码请求，报文返回时解析填充，最后取出结果；
// 单个对象不应被多个并发流程同时复用。
type GetAuction struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetAuctionRequest
	reply      *GetAuctionReply
}

// GetAuctionRequest 是集合竞价查询的请求参数。
type GetAuctionRequest struct {
	Market uint16  // 市场代码（2 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长证券代码，尾部以零字节填充。
	Zero1  uint32  // 协议保留占位，发送时填 0，语义待确认。
	Mode   uint32  // 竞价模式；传 0 时由构造函数回填默认值 3，其余取值语义待确认。
	Zero2  uint32  // 协议保留占位，发送时填 0，语义待确认。
	Start  uint32  // 竞价明细起始偏移（条）。
	Count  uint32  // 请求明细条数；传 0 时按默认 500 处理。
}

// GetAuctionReply 是集合竞价查询的响应数据。
type GetAuctionReply struct {
	Count uint16        // 服务端实际返回的明细条数。
	List  []AuctionData // 解析后的竞价明细列表。
}

// AuctionData 表示一条集合竞价记录，包含成交时间、价格、已匹配量、未匹配量及买卖方向。
// 未匹配量以有符号整数传输：负值表示未匹配单在卖侧，正值表示在买侧；
// 方向标记据此记为 -1（卖单）或 1（买单），未匹配量统一保存绝对值。
type AuctionData struct {
	Time      string  // 成交时间，格式为 `时:分:秒`。
	Price     float64 // 成交价（元）；协议内为 4 字节浮点位模式。
	Matched   uint32  // 已匹配（成交）数量，单位口径以协议为准。
	Unmatched uint32  // 未匹配数量（已取绝对值），单位口径以协议为准。
	Flag      int8    // 方向标记：1 为买单，-1 为卖单。
}

// NewGetAuction 创建并初始化集合竞价查询请求对象。
//
// 参数 `req` 携带查询条件：`Market` 指定市场代码，`Code` 为证券代码，`Mode` 为竞价模式
// （传 0 时回填默认 3），`Start` 为明细起始偏移，`Count` 为请求条数（传 0 时回填默认 500）；
// 传 `nil` 时仅初始化协议骨架，模式取默认 3，条数与偏移保持零值。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetAuction(req *GetAuctionRequest) *GetAuction {
	obj := new(GetAuction)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetAuctionRequest)
	obj.reply = new(GetAuctionReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_AUCTION
	obj.request.Mode = 3
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetAuction) applyRequest(req *GetAuctionRequest) {
	if req.Mode == 0 {
		req.Mode = 3
	}
	if req.Count == 0 {
		req.Count = 500
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数连续编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetAuction) BuildRequest() ([]byte, error) {
	// 包头定长 0x1e 字节，长度校验位 `PkgLen1`、`PkgLen2` 同步置为相同值。
	obj.reqHeader.PkgLen1 = 0x1e
	obj.reqHeader.PkgLen2 = 0x1e

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的集合竞价报文并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 每条记录固定 16 字节：0:2 为时分编码，2:6 为价格，6:10 为已匹配量，
// 10:14 为有符号未匹配量，第 15 字节为秒；记录偏移超出报文长度时返回截断错误。
func (obj *GetAuction) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 报文头：2 字节记录条数。
	if err := binary.Read(bytes.NewBuffer(data[:2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}

	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条记录固定 16 字节，起始偏移跳过 2 字节报文头。
		base := int(i)*16 + 2
		// 越界防护：记录偏移超出报文长度时返回截断错误。
		if base+16 > len(data) {
			return io.ErrUnexpectedEOF
		}

		// 0:2 时分编码为小时乘 60 加分钟，随后按整除与取余还原出时分。
		timeRaw := binary.LittleEndian.Uint16(data[base : base+2])
		// 2:6 为价格的浮点位模式，解码为成交价。
		priceBits := binary.LittleEndian.Uint32(data[base+2 : base+6])
		// 6:10 为已匹配成交量。
		matched := binary.LittleEndian.Uint32(data[base+6 : base+10])
		// unmatched := binary.LittleEndian.Uint32(data[base+10 : base+14])
		// 第 15 字节为秒，与还原出的时分组合成时间字符串。
		second := data[base+15]
		// 10:14 为有符号未匹配量，符号区分买卖方向。
		unmatchedRaw := int32(binary.LittleEndian.Uint32(data[base+10 : base+14]))
		var flag int8 = 1 // 默认买单
		var unmatched uint32
		// 负值归为卖侧并取绝对值；正值归为买侧。
		if unmatchedRaw < 0 {
			flag = -1
			unmatched = uint32(-unmatchedRaw)
		} else {
			flag = 1
			unmatched = uint32(unmatchedRaw)
		}
		obj.reply.List = append(obj.reply.List, AuctionData{
			Time:      fmt.Sprintf("%02d:%02d:%02d", timeRaw/60, timeRaw%60, second),
			Price:     float64(math.Float32frombits(priceBits)),
			Matched:   matched,
			Unmatched: unmatched,
			Flag:      flag,
		})
	}

	return nil
}

// Response 返回解析完成后的集合竞价响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetAuction) Response() *GetAuctionReply {
	return obj.reply
}
