package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// GetHistoryMinuteTimeData 封装指定交易日历史分时（分钟价格、均价、成交量）数据查询的请求与响应处理流程。
type GetHistoryMinuteTimeData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryMinuteTimeDataRequest
	reply      *GetHistoryMinuteTimeDataReply
}

// GetHistoryMinuteTimeDataRequest 是历史分时数据查询的请求参数。
type GetHistoryMinuteTimeDataRequest struct {
	Date   int32   // 交易日期，按 `YYYYMMDD` 正整数传入；构造时内部取负做协议编码。
	Market uint8   // 市场代码（1 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长证券代码，尾部以零字节填充。
}

// GetHistoryMinuteTimeDataReply 是历史分时数据查询的响应数据。
type GetHistoryMinuteTimeDataReply struct {
	Count uint16                  // 服务端返回的分时记录条数。
	List  []HistoryMinuteTimeData // 解析后的历史分时记录列表。
}

// HistoryMinuteTimeData 表示一条历史分时记录，含成交价、均价与成交量。
type HistoryMinuteTimeData struct {
	Price float64 // 成交价（元）；协议内为相对首条的增量，解析时已累加还原。
	Avg   float64 // 均价（元）；协议精度较价格再放大 100 倍，解析时已还原。
	Vol   int     // 该分钟成交量，单位口径以协议为准。
}

// NewGetHistoryMinuteTimeData 创建并初始化历史分时数据查询请求对象。
//
// 参数 `req` 携带 `Date` 查询交易日、`Market` 市场代码与 `Code` 证券代码；
// 传 `nil` 时仅初始化协议骨架。返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetHistoryMinuteTimeData(req *GetHistoryMinuteTimeDataRequest) *GetHistoryMinuteTimeData {
	obj := new(GetHistoryMinuteTimeData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetHistoryMinuteTimeDataRequest)
	obj.reply = new(GetHistoryMinuteTimeDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_HISTORYMINUTETIMEDATE
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetHistoryMinuteTimeData) applyRequest(req *GetHistoryMinuteTimeDataRequest) {
	// 协议用负整数编码历史日期：用户传入正日期时统一取负，避免服务端把它当作最近交易日。
	if req.Date > 0 {
		req.Date = -req.Date
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetHistoryMinuteTimeData) BuildRequest() ([]byte, error) {
	// 包头定长 0x0d 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 0x0d
	obj.reqHeader.PkgLen2 = 0x0d

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的历史分时记录并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 条数头之后为 8 字节保留区；每条记录依次为价格、均价、量三个变长字段。
// 价格与均价是相对首条的差值，先累加成累计值，再按价位单位换算为浮点实数；
// 报文长度不足或变长字段被截断时返回错误。
func (obj *GetHistoryMinuteTimeData) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 最小报文长度校验，避免空或超短数据触发越界。
	if len(data) < 10 {
		return fmt.Errorf("invalid history minute response length: %d", len(data))
	}

	pos := 0
	// 0:2 分时记录条数。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	// 跳过条数后的两个保留字段（共 8 字节），记录区从偏移 10 开始。
	pos += 10

	// 两个累计基准记录首条价格与均价；后续价格、均价均为相对它的增量。
	startPrice := 0
	startAvg := 0
	// 按证券代码取价位单位（0.01 元对应 100、0.001 元对应 1000），用于整数还原。
	unit := baseUnit(string(obj.request.Code[:]))
	for index := uint16(0); index < obj.reply.Count; index++ {
		// 依次读取价格、均价、量三个变长字段；任一字段被截断即报错中止。
		price, ok := readPriceField(data, &pos)
		if !ok {
			return fmt.Errorf("truncated history minute response: pos=%d len=%d", pos, len(data))
		}
		avg, ok := readPriceField(data, &pos)
		if !ok {
			return fmt.Errorf("truncated history minute response: pos=%d len=%d", pos, len(data))
		}
		vol, ok := readPriceField(data, &pos)
		if !ok {
			return fmt.Errorf("truncated history minute response: pos=%d len=%d", pos, len(data))
		}

		// 非首条：把增量叠加到上一条累计价格与均价上，得到本条累计值。
		if startPrice != 0 {
			price += startPrice
		}
		if startAvg != 0 {
			avg += startAvg
		}

		// 价格按价位单位换算；均价协议精度再放大 100 倍，故除以单位乘 100。
		obj.reply.List = append(obj.reply.List, HistoryMinuteTimeData{
			Price: float64(price) / unit,
			Avg:   float64(avg) / (unit * 100),
			Vol:   vol,
		})

		// 首条解析完成后登记累计基准，供后续增量累加。
		if startPrice == 0 {
			startPrice = price
		}
		if startAvg == 0 {
			startAvg = avg
		}
	}

	return nil
}

// readPriceField 安全读取一条变长价格字段。
//
// 参数 `data` 为整段响应报文体，`pos` 为当前读指针，读取成功后向后推进；
// 字段不完整（续字节被截断）时返回值为 0 与 `false`，正常返回解码后的整数与 `true`。
func readPriceField(data []byte, pos *int) (int, bool) {
	i := *pos
	for {
		if i >= len(data) {
			return 0, false
		}
		// 变长编码续位：0x80 表示后面还有字节，扫描至末位为 0 的字节结束。
		if data[i]&0x80 == 0 {
			break
		}
		i++
	}
	return getprice(data, pos), true
}

// Response 返回解析完成后的历史分时响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetHistoryMinuteTimeData) Response() *GetHistoryMinuteTimeDataReply {
	return obj.reply
}
