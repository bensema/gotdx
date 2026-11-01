package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
)

// MACMarketMonitor 按市场、监控模式与阈值筛选并返回异动标的列表。
//
// 使用时先由 NewMACMarketMonitor 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACMarketMonitor struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACMarketMonitorRequest
	reply      *MACMarketMonitorReply
}

// MACMarketMonitorRequest 保存异动监控的市场、起始序号、条数、监控模式与五档阈值参数。
type MACMarketMonitorRequest struct {
	Market    uint16    // 市场代码，两字节，取值见 `types.Market*` 常量。
	Start     uint16    // 起始记录序号（分页偏移），两字节。
	Reserved1 uint16    // 协议保留填充字段，语义待确认。
	Count     uint16    // 请求异动条数，为 0 时默认取 600。
	Reserved2 uint16    // 协议保留填充字段，语义待确认。
	Mode      uint16    // 监控模式，为 0 时默认回填 1，模式编号语义待确认。
	Limits    [5]uint16 // 五档异动阈值，为空时使用默认 `[200, 30, 40, 50, 200]`，各档位含义语义待确认。
}

// MACMarketMonitorReply 保存异动监控响应的记录条数与逐条异动条目。
type MACMarketMonitorReply struct {
	Count uint16                 // 本次返回的异动记录条数，两字节。
	List  []MACMarketMonitorItem // 异动条目列表，按服务端返回顺序排列。
}

// MACMarketMonitorItem 表示一条异动监控记录，包含身份信息、异动类型、时间、
// 由类型决定含义的描述文案与数值，以及四个按类型解释的附属字段。
type MACMarketMonitorItem struct {
	Index       uint16  // 记录序号，两字节。
	Market      uint16  // 市场代码，两字节。
	Code        string  // 证券代码，GBK 解码。
	Name        string  // 证券名称，由报文尾部逗号分隔名称段按顺序回填。
	Time        string  // 异动时间标签，格式 HH:MM:SS。
	Desc        string  // 由异动类型解码得到的描述文案。
	Value       string  // 由异动类型解码得到的数值文本。
	UnusualType uint8   // 异动类型字节，决定负载区的解释方式。
	V1          uint8   // 附属标识字节，含义随异动类型变化，语义待确认。
	V2          float64 // 附属单精度浮点数值一，含义随异动类型变化，语义待确认。
	V3          float64 // 附属单精度浮点数值二，含义随异动类型变化，语义待确认。
	V4          float64 // 附属单精度浮点数值三，含义随异动类型变化，语义待确认。
}

// defaultMACMarketMonitorLimits 返回五档监控阈值的默认值，依次对应不同异动档位；
// 返回值顺序与请求 `Limits` 数组下标一一对应。
func defaultMACMarketMonitorLimits() [5]uint16 {
	return [5]uint16{200, 30, 40, 50, 200}
}

// NewMACMarketMonitor 创建异动监控查询对象；请求参数为空时使用内置默认条数、模式与阈值。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`Market` 为市场代码、`Start` 为起始序号、
// `Count` 为请求条数、`Mode` 为监控模式、`Limits` 为五档阈值；`Count`、`Mode` 为 0 时
// 分别回填默认值 600、1，`Limits` 为空时回填内置默认阈值数组。
func NewMACMarketMonitor(req *MACMarketMonitorRequest) *MACMarketMonitor {
	obj := &MACMarketMonitor{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACMarketMonitorRequest),
		reply:      new(MACMarketMonitorReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACMARKETMONITOR
	obj.request.Count = 600
	obj.request.Mode = 1
	obj.request.Limits = defaultMACMarketMonitorLimits()
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACMarketMonitor) applyRequest(req *MACMarketMonitorRequest) {
	if req.Count == 0 {
		req.Count = 600
	}
	if req.Mode == 0 {
		req.Mode = 1
	}
	if req.Limits == ([5]uint16{}) {
		req.Limits = defaultMACMarketMonitorLimits()
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACMARKETMONITOR` 与请求头封装成完整报文。
func (obj *MACMarketMonitor) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACMARKETMONITOR, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACMarketMonitor) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 2 {
		return fmt.Errorf("invalid mac market monitor response length: %d", len(data))
	}
	// 响应首 2 字节为本次返回的异动记录数，读取。
	if err := binary.Read(bytes.NewReader(data[:2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}

	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条记录固定 32 字节，紧随 2 字节条数头顺序排列。
		base := int(i)*32 + 2
		// 剩余字节不足一条记录时返回意外结束错误，防止越界读取。
		if base+32 > len(data) {
			return io.ErrUnexpectedEOF
		}
		// 可变负载区为偏移 15:28 共 13 字节：首字节为附属标识，其后三段单精度浮点依次为附属数值。
		payload := data[base+15 : base+28]
		v1 := payload[0]
		v2 := float64(math.Float32frombits(binary.LittleEndian.Uint32(payload[1:5])))
		v3 := float64(math.Float32frombits(binary.LittleEndian.Uint32(payload[5:9])))
		v4 := float64(math.Float32frombits(binary.LittleEndian.Uint32(payload[9:13])))
		// 异动类型字节决定负载区应解释为何种描述文案与数值。
		desc, value := unpackUnusualByType(data[base+9], payload)

		// 记录头部：市场（0:2）、GBK 代码（2:8）、序号（11:13）、类型字节（9）；
		// 时间（29）为小时，（30:32）为两字节整数，拆出分钟与秒。
		item := MACMarketMonitorItem{
			Index:       binary.LittleEndian.Uint16(data[base+11 : base+13]),
			Market:      binary.LittleEndian.Uint16(data[base : base+2]),
			Code:        Utf8ToGbk(data[base+2 : base+8]),
			Time:        fmt.Sprintf("%02d:%02d:%02d", int(data[base+29]), int(binary.LittleEndian.Uint16(data[base+30:base+32]))/100, int(binary.LittleEndian.Uint16(data[base+30:base+32]))%100),
			Desc:        desc,
			Value:       value,
			UnusualType: data[base+9],
			V1:          v1,
			V2:          v2,
			V3:          v3,
			V4:          v4,
		}
		obj.reply.List = append(obj.reply.List, item)
	}

	// 记录区总长为 2 字节头加条数乘 32；其后若仍有字节，即为附加名称尾段。
	binaryLength := 2 + int(obj.reply.Count)*32
	if binaryLength >= len(data) {
		return nil
	}

	// 尾段为逗号分隔的证券名称列表，按记录顺序回填到对应条目的名称字段。
	names := strings.Trim(Utf8ToGbk(data[binaryLength:]), ",")
	if names == "" {
		return nil
	}
	parts := strings.Split(names, ",")
	for i := range obj.reply.List {
		if i < len(parts) {
			obj.reply.List[i].Name = parts[i]
		}
	}
	return nil
}

// Response 返回解码完成的异动监控响应；未调用 ParseResponse 前内容为零值。
func (obj *MACMarketMonitor) Response() *MACMarketMonitorReply {
	return obj.reply
}
