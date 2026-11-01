package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// GetHistoryTransactionDataWithTrans 封装带买卖方向（主动买、卖、中性）的历史逐笔成交查询的请求与响应处理流程。
//
// 与不带方向的版本相比，每条成交额外携带独立方向编码，响应头部直接给出昨收价；
// 请求参数复用通用成交请求结构。
type GetHistoryTransactionDataWithTrans struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryTransactionDataRequest
	reply      *GetHistoryTransactionDataWithTransReply
}

// GetHistoryTransactionDataWithTransReply 是带方向历史逐笔成交查询的响应数据。
type GetHistoryTransactionDataWithTransReply struct {
	Count    uint16                            // 服务端返回的成交记录条数。
	PreClose float64                           // 昨收价（元）；协议内为 4 字节浮点位模式。
	List     []HistoryTransactionDataWithTrans // 解析后的带方向历史逐笔成交列表。
}

// HistoryTransactionDataWithTrans 表示一条带成交方向的历史逐笔成交记录。
type HistoryTransactionDataWithTrans struct {
	Time   time.Time // 成交时间，由报文中的编码时分与请求日期拼接为本地时间。
	Price  float64   // 成交价（元）；协议内为累计差值，按证券价位单位还原。
	Vol    int       // 本笔成交量，单位口径以协议为准。
	Num    int       // 本笔汇总成交笔数。
	Action string    // 成交方向文本：买、卖、中性；未知编码以数字字符串原样保留。
}

// NewGetHistoryTransactionDataWithTrans 创建并初始化带方向历史逐笔成交查询请求对象。
//
// 参数 `req` 复用通用成交请求结构，携带 `Date` 查询交易日、`Market` 市场代码、
// `Code` 证券代码、`Start` 起始偏移与 `Count` 请求条数；传 `nil` 时仅初始化协议骨架。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetHistoryTransactionDataWithTrans(req *GetHistoryTransactionDataRequest) *GetHistoryTransactionDataWithTrans {
	obj := &GetHistoryTransactionDataWithTrans{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(GetHistoryTransactionDataRequest),
		reply:      new(GetHistoryTransactionDataWithTransReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_TRANSACTIONDATA_TRANS
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetHistoryTransactionDataWithTrans) applyRequest(req *GetHistoryTransactionDataRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetHistoryTransactionDataWithTrans) BuildRequest() ([]byte, error) {
	// 包头定长 0x12 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 0x12
	obj.reqHeader.PkgLen2 = 0x12
	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	err := binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的带方向逐笔成交记录并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 0:2 条数，2:6 昨收价；其后每条记录依次为编码时间、价格差值、成交量、笔数与 2 字节方向编码；
// 报文头部长度不足、时间拼接失败或方向编码字节缺失时返回错误。
func (obj *GetHistoryTransactionDataWithTrans) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 6 字节（条数 2 加昨收价 4），不足视为截断。
	if len(data) < 6 {
		return fmt.Errorf("invalid history transaction with trans response length: %d", len(data))
	}
	// 0:2 成交笔数。
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	// 2:6 昨收价。
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[2:6])))
	pos := 6
	// 累计成交价；协议每条给出的是价格差值。
	lastPrice := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 编码时分，与请求日期拼接为本地成交时间。
		hour, minute := gettime(data, &pos)
		// 价格差值（变长编码）。
		priceDiff := getprice(data, &pos)
		// 成交量。
		vol := getprice(data, &pos)
		// 成交笔数（汇总笔数）。
		num := getprice(data, &pos)
		// 边界防护：方向编码占 2 字节，剩余字节不足时中止并返回当前序号。
		if pos+2 > len(data) {
			return fmt.Errorf("invalid history transaction with trans item %d", i)
		}
		// 2 字节方向编码：0 买、1 卖、2 中性。
		actionCode := binary.LittleEndian.Uint16(data[pos : pos+2])
		pos += 2
		lastPrice += priceDiff
		nowDate := fmt.Sprintf("%d", obj.request.Date)
		hourMinute := fmt.Sprintf("%02d:%02d", hour, minute)
		nowTime, err := time.ParseInLocation("2006010215:04", nowDate+hourMinute, time.Local)
		if err != nil {
			return err
		}
		item := HistoryTransactionDataWithTrans{
			Time:  nowTime,
			Price: float64(lastPrice) / baseUnit(string(obj.request.Code[:])),
			Vol:   vol,
			Num:   num,
		}
		// 方向编码映射为文本动作；未知编码原样以数字字符串保留。
		switch actionCode {
		case 0:
			item.Action = "BUY"
		case 1:
			item.Action = "SELL"
		case 2:
			item.Action = "NEUTRAL"
		default:
			item.Action = fmt.Sprintf("%d", actionCode)
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	return nil
}

// Response 返回解析完成后的带方向历史逐笔成交响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetHistoryTransactionDataWithTrans) Response() *GetHistoryTransactionDataWithTransReply {
	return obj.reply
}
