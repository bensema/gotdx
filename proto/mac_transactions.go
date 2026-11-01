package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// MACTransactions 分页查询单只标的当日的逐笔成交记录。
//
// 使用时先由 NewMACTransactions 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACTransactions struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACTransactionsRequest
	reply      *MACTransactionsReply
}

// MACTransactionsRequest 保存逐笔成交查询的市场、代码、交易日、起始序号与条数参数。
type MACTransactionsRequest struct {
	Market    uint16   // 市场代码，两字节，取值见 `types.Market*` 常量。
	Code      [22]byte // 证券代码，GBK 编码定长 22 字节，不足补空字符。
	QueryDate uint32   // 查询交易日打包整数（YYYYMMDD），传 0 表示取最新交易日。
	Start     uint32   // 起始记录序号（分页偏移），四字节。
	Count     uint16   // 请求成交条数，为 0 时默认取 1000。
	Reserved  [10]byte // 协议保留填充字节，填零。
}

// MACTransactionsReply 保存逐笔成交响应的标的标识、计数、起始序号、成交总数与逐条记录。
type MACTransactionsReply struct {
	Market    uint16               // 市场代码，两字节。
	Code      string               // 证券代码，GBK 解码。
	QueryDate uint32               // 回显的查询交易日打包整数，四字节。
	Count     uint16               // 本次返回的成交条数，两字节。
	Start     uint32               // 回显的起始记录序号，四字节。
	Total     uint32               // 当日成交总数，四字节。
	List      []MACTransactionItem // 逐笔成交记录列表，按时间顺序排列。
}

// MACTransactionItem 表示一条逐笔成交记录，包含时间标签、成交价、成交量、成交笔数与买卖方向标记。
type MACTransactionItem struct {
	Time       string  // 成交时间标签，格式 HH:MM:SS，由当日秒数拆分得到。
	Price      float64 // 成交价，单精度浮点还原。
	Vol        uint32  // 成交量，四字节无符号整数。
	TradeCount uint32  // 成交笔数，四字节无符号整数。
	BuyOrSell  uint16  // 买卖方向标记，两字节，取值含义语义待确认。
}

// NewMACTransactions 创建逐笔成交查询对象；请求参数为空时使用内置默认条数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`Market` 为市场代码、`Code` 为证券代码、
// `QueryDate` 为查询交易日（传 0 取最新交易日）、`Start` 为起始序号、`Count` 为请求条数；
// `Count` 为 0 时回填默认值 1000。
func NewMACTransactions(req *MACTransactionsRequest) *MACTransactions {
	obj := &MACTransactions{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACTransactionsRequest),
		reply:      new(MACTransactionsReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACTRANSACTIONS
	obj.request.Count = 1000
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACTransactions) applyRequest(req *MACTransactionsRequest) {
	if req.Count == 0 {
		req.Count = 1000
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACTRANSACTIONS` 与请求头封装成完整报文。
func (obj *MACTransactions) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACTRANSACTIONS, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACTransactions) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 39 {
		return fmt.Errorf("invalid mac transactions response length: %d", len(data))
	}

	// 头部布局：市场（0:2）、GBK 代码（2:24）、交易日（24:28）、条数（29:31）、起始（31:35）、总数（35:39）。
	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:24])
	obj.reply.QueryDate = binary.LittleEndian.Uint32(data[24:28])
	obj.reply.Count = binary.LittleEndian.Uint16(data[29:31])
	obj.reply.Start = binary.LittleEndian.Uint32(data[31:35])
	obj.reply.Total = binary.LittleEndian.Uint32(data[35:39])

	pos := 39
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 剩余字节不足一条记录时中止并报错，防止越界读取。
		if pos+18 > len(data) {
			return fmt.Errorf("invalid mac transactions item %d", i)
		}
		// 记录布局：当日秒数（0:4）、成交价（4:8）、成交量（8:12）、笔数（12:16）、方向（16:18）；价格为单精度浮点。
		seconds := binary.LittleEndian.Uint32(data[pos : pos+4])
		obj.reply.List = append(obj.reply.List, MACTransactionItem{
			// 秒数拆分为时、分、秒，生成成交时间标签。
			Time:       time.Date(0, 1, 1, int(seconds/3600)%24, int((seconds%3600)/60), int(seconds%60), 0, time.Local).Format("15:04:05"),
			Price:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))),
			Vol:        binary.LittleEndian.Uint32(data[pos+8 : pos+12]),
			TradeCount: binary.LittleEndian.Uint32(data[pos+12 : pos+16]),
			BuyOrSell:  binary.LittleEndian.Uint16(data[pos+16 : pos+18]),
		})
		pos += 18
	}
	return nil
}

// Response 返回解码完成的逐笔成交响应；未调用 ParseResponse 前内容为零值。
func (obj *MACTransactions) Response() *MACTransactionsReply {
	return obj.reply
}
