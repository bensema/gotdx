package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

// GetHistoryOrders 封装指定交易日历史委托（分时挂单档位）数据查询的请求与响应处理流程。
type GetHistoryOrders struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryOrdersRequest
	reply      *GetHistoryOrdersReply
}

// GetHistoryOrdersRequest 是历史委托数据查询的请求参数。
type GetHistoryOrdersRequest struct {
	Date   uint32  // 交易日期，按 `YYYYMMDD` 整数编码（4 字节）。
	Market uint8   // 市场代码（1 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长证券代码，尾部以零字节填充。
}

// GetHistoryOrdersReply 是历史委托数据查询的响应数据。
type GetHistoryOrdersReply struct {
	Count    uint16             // 服务端返回的档位记录条数。
	PreClose float64            // 昨收价（元）；协议内为 4 字节浮点位模式。
	List     []HistoryOrderData // 解析后的历史委托档位列表。
}

// HistoryOrderData 表示一条历史委托档位记录，含价格与挂单量。
type HistoryOrderData struct {
	Price   float64 // 档位价格（元）；协议内为累计差值，解析时累加后除以 100 还原。
	Unknown int     // 未确认字段，语义待确认。
	Vol     int     // 该档位挂单量，单位口径以协议为准。
}

// NewGetHistoryOrders 创建并初始化历史委托数据查询请求对象。
//
// 参数 `req` 携带 `Date` 查询交易日、`Market` 市场代码与 `Code` 证券代码；
// 传 `nil` 时仅初始化协议骨架。返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetHistoryOrders(req *GetHistoryOrdersRequest) *GetHistoryOrders {
	obj := new(GetHistoryOrders)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetHistoryOrdersRequest)
	obj.reply = new(GetHistoryOrdersReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_HISTORYORDERS
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetHistoryOrders) applyRequest(req *GetHistoryOrdersRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetHistoryOrders) BuildRequest() ([]byte, error) {
	// 包头定长 0x0d 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 0x0d
	obj.reqHeader.PkgLen2 = 0x0d

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的历史委托档位记录并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 0:2 条数，2:6 昨收价，其后为变长编码的档位记录序列。
func (obj *GetHistoryOrders) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	// 0:2 档位记录条数。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2
	// 2:6 昨收价。
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4

	// 累计档位价格；协议每条给的是价格差值而非绝对价格。
	lastPrice := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 价格差值（变长编码）。
		priceRaw := getprice(data, &pos)
		// 保留/未确认字段，语义待确认。
		unknown := getprice(data, &pos)
		// 该档位挂单量。
		vol := getprice(data, &pos)
		// 累加得到本档绝对价位，再除以 100 还原为实际价格。
		lastPrice += priceRaw
		obj.reply.List = append(obj.reply.List, HistoryOrderData{
			Price:   float64(lastPrice) / 100.0,
			Unknown: unknown,
			Vol:     vol,
		})
	}

	return nil
}

// Response 返回解析完成后的历史委托响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetHistoryOrders) Response() *GetHistoryOrdersReply {
	return obj.reply
}
