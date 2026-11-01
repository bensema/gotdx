package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
)

// GetHistoryTransactionData 封装指定交易日历史逐笔成交（分笔）数据查询的请求与响应处理流程。
type GetHistoryTransactionData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryTransactionDataRequest
	reply      *GetHistoryTransactionDataReply

	contentHex string
}

// GetHistoryTransactionDataRequest 是历史逐笔成交查询的请求参数。
type GetHistoryTransactionDataRequest struct {
	Date   uint32  // 交易日期，按 `YYYYMMDD` 整数编码（4 字节）。
	Market uint16  // 市场代码（2 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长证券代码，尾部以零字节填充。
	Start  uint16  // 成交记录起始偏移（条，2 字节）。
	Count  uint16  // 请求成交记录条数（2 字节）。
}

// GetHistoryTransactionDataReply 是历史逐笔成交查询的响应数据。
type GetHistoryTransactionDataReply struct {
	Count uint16                   // 服务端返回的成交记录条数。
	List  []HistoryTransactionData // 解析后的历史逐笔成交列表。
}

// HistoryTransactionData 表示一条历史逐笔成交记录，包含时间、价格、成交量与买卖方向。
type HistoryTransactionData struct {
	Time      time.Time // 成交时间，由报文中的编码时分与请求日期拼接为本地时间。
	Price     float64   // 成交价（元）；协议内为累计差值，按证券价位单位还原。
	Vol       int       // 本笔成交量，单位口径以协议为准。
	Num       int       // 笔数或委托笔数；当前报文解析未填充该字段，语义待确认。
	BuyOrSell int       // 买卖方向原始编码：0 中性，其余为买或卖。
	Action    string    // 方向编码映射后的文本，取值为买、卖、中性。
	Unknown   int       // 末尾保留字段，语义待确认。
}

// NewGetHistoryTransactionData 创建并初始化历史逐笔成交查询请求对象。
//
// 参数 `req` 携带 `Date` 查询交易日、`Market` 市场代码、`Code` 证券代码、
// `Start` 起始偏移与 `Count` 请求条数；传 `nil` 时仅初始化协议骨架。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetHistoryTransactionData(req *GetHistoryTransactionDataRequest) *GetHistoryTransactionData {
	obj := new(GetHistoryTransactionData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetHistoryTransactionDataRequest)
	obj.reply = new(GetHistoryTransactionDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	//obj.reqHeader.PkgLen1  =
	//obj.reqHeader.PkgLen2  =
	obj.reqHeader.Method = KMSG_HISTORYTRANSACTIONDATA
	//obj.reqHeader.Method = KMSG_MINUTETIMEDATA
	obj.contentHex = ""
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetHistoryTransactionData) applyRequest(req *GetHistoryTransactionDataRequest) {
	obj.request = req
}

// BuildRequest 将请求头、请求参数及可选附加十六进制内容编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出或附加载荷解码失败时返回错误。
func (obj *GetHistoryTransactionData) BuildRequest() ([]byte, error) {
	// 包头定长 0x12 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 0x12
	obj.reqHeader.PkgLen2 = 0x12

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	// 解码可选附加十六进制载荷并追加到参数之后（当前默认无附加内容）。
	b, err := hex.DecodeString(obj.contentHex)
	buf.Write(b)

	//b, err := hex.DecodeString(obj.contentHex)
	//buf.Write(b)

	//err = binary.Write(buf, binary.LittleEndian, uint16(len(obj.stocks)))

	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的历史逐笔成交记录并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 0:2 条数，2:6 为保留区；其后每条成交为变长编码字段序列；
// 时间拼接失败或记录字段读取异常时返回错误。
func (obj *GetHistoryTransactionData) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	// 0:2 成交笔数。
	err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	// 条数后固定跳过 4 字节保留区，记录区从偏移 6 开始。
	pos += 6

	// 累计成交价；协议每条给出的是价格差值而非绝对价格。
	lastprice := 0
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := HistoryTransactionData{}
		// 读取编码的时分，与请求日期拼接为本地时间。
		hour, minute := gettime(data, &pos)
		nowDate := fmt.Sprintf("%d", obj.request.Date)
		hourMinute := fmt.Sprintf("%02d:%02d", hour, minute)
		nowTime, err := time.ParseInLocation("2006010215:04", nowDate+hourMinute, time.Local)
		if err != nil {
			return err
		}
		ele.Time = nowTime
		// 价格差值（变长编码）。
		priceraw := getprice(data, &pos)
		// 成交量。
		ele.Vol = getprice(data, &pos)
		// 买卖方向原始编码（0 中性，其余为买或卖）。
		ele.BuyOrSell = getprice(data, &pos)
		// 方向编码映射为买、卖、中性文本。
		ele.Action = actionFromCode(ele.BuyOrSell)
		// 末尾保留字段，语义待确认。
		ele.Unknown = getprice(data, &pos)

		// 累加得到本笔累计价位，再按证券价位单位换算为实际成交价格。
		lastprice = lastprice + priceraw
		ele.Price = float64(lastprice) / baseUnit(string(obj.request.Code[:]))
		obj.reply.List = append(obj.reply.List, ele)
	}
	return err
}

// Response 返回解析完成后的历史逐笔成交响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetHistoryTransactionData) Response() *GetHistoryTransactionDataReply {
	return obj.reply
}
