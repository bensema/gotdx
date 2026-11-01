package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// GetTransactionData 表示一次获取当日分笔成交明细协议的完整流程对象。
//
// 该对象按证券代码分页返回当日逐笔成交记录。零值不可直接使用，必须通过
// NewGetTransactionData 构造；对象绑定单次查询流程，不支持多协程并发复用。
type GetTransactionData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetTransactionDataRequest
	reply      *GetTransactionDataReply

	contentHex string
}

// GetTransactionDataRequest 是 GetTransactionData 分笔成交查询使用的请求参数。
type GetTransactionDataRequest struct {
	Market uint16  // 市场代码，`uint16`。
	Code   [6]byte // 6 字节定长证券代码。
	Start  uint16  // 起始偏移，`uint16`，从 0 开始按页递增。
	Count  uint16  // 请求条数，`uint16`。
}

// GetTransactionDataReply 是 GetTransactionData 分笔成交查询解析完成后的响应数据。
type GetTransactionDataReply struct {
	Count uint16            // 返回笔数，`uint16`。
	List  []TransactionData // 当日逐笔成交记录列表，长度与 `Count` 一致。
}

// TransactionData 表示一笔分笔成交记录，包含时间、价格、量、笔数与买卖方向。
type TransactionData struct {
	Time      string  // 成交时间，格式为时:分，由当日分钟数拆分。
	Price     float64 // 成交价，单位为元，按代码前缀缩放单位还原。
	Vol       int     // 成交量（手），变长整数解码。
	Num       int     // 笔数或委托笔数，变长整数解码。
	Trans     int     // 委托笔数字段，语义待确认。
	BuyOrSell int     // 买卖方向原始标记码，变长整数解码。
	Action    string  // 买卖方向文字，取值为买、卖或中性，由 `BuyOrSell` 映射得到。
	Unknown   int     // 未确认字段，变长整数解码，语义待确认。
}

// NewGetTransactionData 构造分笔成交查询对象。
//
// 参数 `req` 指定市场 `Market`、证券代码 `Code`、起始偏移 `Start` 与请求条数
// `Count`；`req` 为 `nil` 时使用空请求参数。返回的对象可直接调用 `BuildRequest`
// 与 `ParseResponse`。
func NewGetTransactionData(req *GetTransactionDataRequest) *GetTransactionData {
	obj := new(GetTransactionData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetTransactionDataRequest)
	obj.reply = new(GetTransactionDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	//obj.reqHeader.PkgLen1  =
	//obj.reqHeader.PkgLen2  =
	obj.reqHeader.Method = KMSG_TRANSACTIONDATA
	//obj.reqHeader.Method = KMSG_MINUTETIMEDATA
	obj.contentHex = ""
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetTransactionData) applyRequest(req *GetTransactionDataRequest) {
	obj.request = req
}

// BuildRequest 将请求头、请求参数及可选附加负载编码为协议报文。
//
// 返回的字节切片可直接写入连接；`PkgLen1` 与 `PkgLen2` 双写为同一负载长度，
// 缓冲区写入失败时返回错误。
func (obj *GetTransactionData) BuildRequest() ([]byte, error) {
	// 负载 14 字节（市场 2 + 代码 6 + 偏移 2 + 条数 2 + 保留 2），包头长度双写为 0x0e。
	obj.reqHeader.PkgLen1 = 0x0e
	obj.reqHeader.PkgLen2 = 0x0e

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	// 解码可选的十六进制扩展负载并追加到报文尾部；上层一般留空。
	b, err := hex.DecodeString(obj.contentHex)
	buf.Write(b)

	//b, err := hex.DecodeString(obj.contentHex)
	//buf.Write(b)

	//err = binary.Write(buf, binary.LittleEndian, uint16(len(obj.stocks)))

	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的分笔成交明细报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；解码失败时返回错误。
func (obj *GetTransactionData) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	// 头部 2 字节：成交笔数。
	err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	pos += 2

	lastprice := 0
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := TransactionData{}
		// 时间为自当日零点起的分钟数，`gettime` 拆分为小时与分钟并推进 2 字节。
		hour, minute := gettime(data, &pos)
		ele.Time = fmt.Sprintf("%02d:%02d", hour, minute)
		// 价格差值：累加后作为本笔成交价。
		priceraw := getprice(data, &pos)
		ele.Vol = getprice(data, &pos)
		ele.Num = getprice(data, &pos)
		ele.Trans = ele.Num
		// 买卖方向原始码，再映射为文字方向。
		ele.BuyOrSell = getprice(data, &pos)
		ele.Action = actionFromCode(ele.BuyOrSell)
		// 累计差值得到当前成交价，按代码前缀缩放为元。
		lastprice += priceraw
		ele.Price = float64(lastprice) / baseUnit(string(obj.request.Code[:]))
		// 尾部未知字段，读取后保留原值，语义待确认。
		ele.Unknown = getprice(data, &pos)
		obj.reply.List = append(obj.reply.List, ele)
	}
	return err
}

// Response 返回已解析完成的分笔成交响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetTransactionData) Response() *GetTransactionDataReply {
	return obj.reply
}

// actionFromCode 将服务端买卖方向码映射为可读字符串。
//
// 参数 `code` 为原始方向码：0 为买、1 为卖、2 为中性；返回对应方向文字，其余
// 未知编码原样返回数字字符串。
func actionFromCode(code int) string {
	switch code {
	case 0:
		return "BUY"
	case 1:
		return "SELL"
	case 2:
		return "NEUTRAL"
	default:
		return fmt.Sprintf("%d", code)
	}
}
