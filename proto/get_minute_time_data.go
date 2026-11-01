package proto

import (
	"bytes"
	"encoding/binary"
)

// GetMinuteTimeData 表示一次当日分时数据查询协议的完整流程对象。
//
// 该对象持有请求头、请求参数与响应数据；零值不可直接使用，必须通过
// NewGetMinuteTimeData 构造。对象绑定单次查询流程，不支持多协程并发复用。
type GetMinuteTimeData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetMinuteTimeDataRequest
	reply      *GetMinuteTimeDataReply
}

// GetMinuteTimeDataRequest 是 GetMinuteTimeData 查询使用的请求参数。
type GetMinuteTimeDataRequest struct {
	Market uint16  // 市场代码。
	Code   [6]byte // 证券代码。
	Start  uint16  // 起始偏移。
	Count  uint16  // 请求条数。
}

// GetMinuteTimeDataReply 是 GetMinuteTimeData 查询解析完成后的响应数据。
type GetMinuteTimeDataReply struct {
	Count uint16           // 返回条数。
	List  []MinuteTimeData // 当日分时采样点列表。
}

// MinuteTimeData 表示一个分时采样点，包含成交价、均价与成交量。
type MinuteTimeData struct {
	Price float64 // 成交价，单位为元。
	Avg   float64 // 均价，单位为元。
	Vol   int     // 成交量。
}

// NewGetMinuteTimeData 构造当日分时数据查询对象；`req` 为 `nil` 时使用空请求参数，返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetMinuteTimeData(req *GetMinuteTimeDataRequest) *GetMinuteTimeData {
	obj := new(GetMinuteTimeData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetMinuteTimeDataRequest)
	obj.reply = new(GetMinuteTimeDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_MINUTETIMEDATA
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetMinuteTimeData) applyRequest(req *GetMinuteTimeDataRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文；缓冲区写入失败时返回错误。
func (obj *GetMinuteTimeData) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0e
	obj.reqHeader.PkgLen2 = 0x0e

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的当日分时报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；报文长度不足或解码失败
// 时返回错误。
func (obj *GetMinuteTimeData) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	pos := 0
	var ignored uint16
	// 头部 2 字节：的采样条数。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2
	// 第二个 `uint16` 为协议保留字段，读取后直接丢弃。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ignored); err != nil {
		return err
	}
	pos += 2

	startPrice := 0
	startAvg := 0
	// 按证券代码前缀确定价格缩放单位，股票为 100，部分 `ETF` 为 1000。
	unit := baseUnit(string(obj.request.Code[:]))
	for index := uint16(0); index < obj.reply.Count; index++ {
		// 依次读取价格差值、均价差值、成交量，`pos` 由 `getprice` 自动推进。
		price := getprice(data, &pos)
		avg := getprice(data, &pos)
		vol := getprice(data, &pos)

		// 服务端下发相对首条样本的差值；非首条样本将差值叠加到首条基准上还原绝对值。
		if startPrice != 0 {
			price += startPrice
		}
		if startAvg != 0 {
			avg += startAvg
		}

		obj.reply.List = append(obj.reply.List, MinuteTimeData{
			Price: float64(price) / unit,
			Avg:   float64(avg) / (unit * 100),
			Vol:   vol,
		})

		// 首条样本解析完成后记录基准价格与均价，供后续差值累加使用。
		if startPrice == 0 {
			startPrice = price
		}
		if startAvg == 0 {
			startAvg = avg
		}
	}

	return nil
}

// Response 返回已解析完成的分时数据响应对象；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetMinuteTimeData) Response() *GetMinuteTimeDataReply {
	return obj.reply
}
