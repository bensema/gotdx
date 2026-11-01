package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
)

// RawDataReply 是原始试验性协议的响应数据，原样保留服务端返回的字节及其十六进制文本。
// 不做任何字段级解析，供上层试验、抓包调试与人工对照协议使用。
type RawDataReply struct {
	Length int    // 响应字节长度，等于 `Data` 的字节数。
	Data   []byte // 原始响应字节，独立拷贝，不引用服务端复用的缓冲区。
	Hex    string // 原始响应字节的小写十六进制文本。
}

// RawMainProtocol 表示通用原始主协议，按指定消息号发送自定义字节流；响应不做字段级解析，
// 原样保留供上层试验与调试使用。
type RawMainProtocol struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *RawDataReply
	method     uint16
	payload    []byte
	generic    bool
}

// newRawMainProtocol 构造普通格式的原始主协议对象。
//
// 参数 `method` 为协议消息号；`payload` 为待发送的原始载荷字节，内部做独立拷贝避免引用外部切片。
// 返回的实例零值不可直接使用。
func newRawMainProtocol(method uint16, payload []byte) *RawMainProtocol {
	obj := &RawMainProtocol{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(RawDataReply),
		method:     method,
		payload:    append([]byte(nil), payload...),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = method
	return obj
}

// newGenericRawMainProtocol 在普通构造基础上切换为通用报文格式。
//
// 参数 `method` 为协议消息号；`payload` 为待发送的原始载荷字节；返回实例使用通用报文头拼装请求。
func newGenericRawMainProtocol(method uint16, payload []byte) *RawMainProtocol {
	obj := newRawMainProtocol(method, payload)
	obj.generic = true
	return obj
}

// BuildRequest 按普通或通用报文格式编码请求头与原始载荷字节流。
//
// 返回值第一项为编码后的请求字节流；请求头序列化失败时第二项返回非空错误。
func (obj *RawMainProtocol) BuildRequest() ([]byte, error) {
	if obj.generic {
		return buildGenericRequest(0x0c, 0, 0x01, obj.method, obj.payload)
	}
	// 报文长度为载荷长度加 2 字节消息号，两个长度字段保持一致，均为`uint16`。
	obj.reqHeader.PkgLen1 = uint16(2 + len(obj.payload))
	obj.reqHeader.PkgLen2 = obj.reqHeader.PkgLen1
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	_, err := buf.Write(obj.payload)
	return buf.Bytes(), err
}

// ParseResponse 原样保存响应字节并生成对应十六进制文本，不做字段级解析。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；`Data` 通过独立切片拷贝，避免
// 引用服务端复用的缓冲区；本方法不返回错误。
func (obj *RawMainProtocol) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	obj.reply.Length = len(data)
	obj.reply.Data = append([]byte(nil), data...)
	obj.reply.Hex = hex.EncodeToString(data)
	return nil
}

// Response 返回原始试验性协议的响应数据。
func (obj *RawMainProtocol) Response() *RawDataReply {
	return obj.reply
}

// NewTodoB 构造 TODOB 试验性原始协议请求，携带固定编码的十六进制报文；本方法不接受入参。
//
// 报文由多段固定十六进制串拼接而成，经 `mustDecodeHex` 解码；报文内容与协议用途语义待确认。
// 返回构造完成的原始协议实例。
func NewTodoB() *RawMainProtocol {
	return newRawMainProtocol(KMSG_TODOB, mustDecodeHex(
		"e53878ee8bd8dbb8"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"901a1266bd9f62d9"+
			"6810db2bdf3e50a1"+
			"9e93269128ddf91f"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"27fd50435e32ca0d"+
			"8872a27c327343f1"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"7c8810a76fd73daf",
	))
}

// NewTodoFDE 构造 TODOFDE（`KMSG_TODOFDE`）试验性原始协议请求，载荷为空；本方法不接受入参，
// 具体用途语义待确认。返回构造完成的原始协议实例。
func NewTodoFDE() *RawMainProtocol { return newRawMainProtocol(KMSG_TODOFDE, nil) }

// NewClient264B 构造 264B 试验性客户端原始协议请求（通用格式）；本方法不接受入参，用途语义待确认。
// 返回构造完成的原始协议实例。
func NewClient264B() *RawMainProtocol {
	return newGenericRawMainProtocol(KMSG_CLIENT264B, []byte{0x00, 0x64})
}

// NewClient26AE 构造 26AE 试验性客户端原始协议请求（通用格式）；本方法不接受入参，用途语义待确认。
// 返回构造完成的原始协议实例。
func NewClient26AE() *RawMainProtocol {
	return newGenericRawMainProtocol(KMSG_CLIENT26AE, []byte{0x00, 0x64})
}

// NewClient26B1 构造 26B1 试验性客户端原始协议请求（通用格式）；本方法不接受入参，用途语义待确认。
// 返回构造完成的原始协议实例。
func NewClient26B1() *RawMainProtocol {
	return newGenericRawMainProtocol(KMSG_CLIENT26B1, []byte{0x00, 0x64, 0x00})
}

// NewClient26AC 构造 26AC 试验性客户端信息上报原始协议请求（通用格式）；本方法不接受入参，
// 用途语义待确认。返回构造完成的原始协议实例。
func NewClient26AC() *RawMainProtocol {
	payload := new(bytes.Buffer)
	// 两个参数（0、100）与一个`uint16` 编号（1234）。
	_ = binary.Write(payload, binary.LittleEndian, uint8(0))
	_ = binary.Write(payload, binary.LittleEndian, uint8(100))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1234))
	// 24 字节协议保留零填充。
	payload.Write(make([]byte, 24))
	// 示例内网 IP `192.168.0.1`，四个字节依次写入。
	payload.Write([]byte{192, 168, 0, 1})
	// 6 字节 MAC 占位内容，依次写入。
	mac := [6]byte{'m', 'a', 'c', 'a', 'd', 'd'}
	_ = binary.Write(payload, binary.LittleEndian, mac)
	// 末尾三个`uint16` 参数（40、1864、0），语义待确认。
	_ = binary.Write(payload, binary.LittleEndian, uint16(40))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1864))
	_ = binary.Write(payload, binary.LittleEndian, uint16(0))
	return newGenericRawMainProtocol(KMSG_CLIENT26AC, payload.Bytes())
}

// NewClient26AD 构造 26AD 试验性客户端身份与令牌上报原始协议请求（通用格式）；本方法不接受
// 入参，用途语义待确认。返回构造完成的原始协议实例。
func NewClient26AD() *RawMainProtocol {
	payload := new(bytes.Buffer)
	// 两个参数（0、100）。
	_ = binary.Write(payload, binary.LittleEndian, uint8(0))
	_ = binary.Write(payload, binary.LittleEndian, uint8(100))
	// 16 字节 `TDXW` 标签占位内容。
	tag := [16]byte{'T', 'D', 'X', 'W'}
	_ = binary.Write(payload, binary.LittleEndian, tag)
	// 24 字节协议保留零填充。
	payload.Write(make([]byte, 24))
	// 示例内网 IP `192.168.0.1`，四个字节依次写入。
	payload.Write([]byte{192, 168, 0, 1})
	// 6 字节 MAC 占位内容。
	mac := [6]byte{'m', 'a', 'c', 'a', 'd', 'd'}
	_ = binary.Write(payload, binary.LittleEndian, mac)
	tag2 := [16]byte{'T', 'D', 'X', 'W'}
	_ = binary.Write(payload, binary.LittleEndian, tag2)
	// 编号与状态类`uint16`/参数，语义待确认。
	_ = binary.Write(payload, binary.LittleEndian, uint16(40))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1864))
	_ = binary.Write(payload, binary.LittleEndian, uint16(0))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1234))
	_ = binary.Write(payload, binary.LittleEndian, uint8(1))
	// 17 字节固定令牌占位内容。
	token := [17]byte{'O', 'P', 'r', '2', 'I', 'H', 'Z', '5', 'r', '3', 'l', 'u', 'K', '9', 'K', 'a'}
	_ = binary.Write(payload, binary.LittleEndian, token)
	_ = binary.Write(payload, binary.LittleEndian, uint8(2))
	_ = binary.Write(payload, binary.LittleEndian, uint8(1))
	return newGenericRawMainProtocol(KMSG_CLIENT26AD, payload.Bytes())
}

// GetSecurityFeature452 表示 452 消息号试验性证券特征批量查询协议，返回证券特征记录列表。
type GetSecurityFeature452 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityFeature452Request
	reply      *GetSecurityFeature452Reply
}

// GetSecurityFeature452Request 保存 452 试验性证券特征查询的起始偏移与条数参数。
type GetSecurityFeature452Request struct {
	Start uint32 // 起始偏移条数，`uint32`，0 表示从第一条开始。
	Count uint32 // 请求条数，`uint32`，传 0 时构造器补默认值 2000。
	One   uint32 // 协议保留开关，`uint32`，传 0 时构造器补默认值 1，语义待确认。
	Zero  uint16 // 保留置零字段，`uint16`，语义待确认。
}

// GetSecurityFeature452Reply 保存解码完成的 452 证券特征查询响应结果。
type GetSecurityFeature452Reply struct {
	Count uint16                   // 返回条数，`uint16`。
	List  []SecurityFeature452Item // 解码出的特征条目列表。
}

// SecurityFeature452Item 表示一条 452 试验性特征记录，两个数值字段的确切含义语义待确认。
type SecurityFeature452Item struct {
	Market uint8   // 市场代码。
	Code   string  // 证券代码，由条目内`uint32` 编号十进制格式化得到。
	P1     float64 // 特征数值一，由 `float32` 还原，语义待确认。
	P2     float64 // 特征数值二，由 `float32` 还原，语义待确认。
}

// NewGetSecurityFeature452 构造 452 特征查询协议实例。
//
// 参数 `req` 指定起始偏移、请求条数与保留开关；传入非空对象时覆盖默认参数，传 `nil` 时使用默认值
// （条数 2000、开关 1）。返回的实例零值不可直接使用。
func NewGetSecurityFeature452(req *GetSecurityFeature452Request) *GetSecurityFeature452 {
	obj := &GetSecurityFeature452{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &GetSecurityFeature452Request{Count: 2000, One: 1},
		reply:      new(GetSecurityFeature452Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYFEATURE452
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetSecurityFeature452) applyRequest(req *GetSecurityFeature452Request) {
	if req.Count == 0 {
		req.Count = 2000
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 按 452 协议固定报文长度写请求头，再追加请求参数。
//
// 返回值第一项为编码后的请求字节流；请求头或参数序列化失败时第二项返回非空错误。
func (obj *GetSecurityFeature452) BuildRequest() ([]byte, error) {
	// 协议固定两个长度字段均为 16 字节。
	obj.reqHeader.PkgLen1 = 16
	obj.reqHeader.PkgLen2 = 16
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	err := binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 从服务端响应字节流解码 452 特征记录并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；缺少 2 字节条数头部或特征条目
// 越界时返回非空错误，解码成功返回 `nil`。
func (obj *GetSecurityFeature452) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 2 字节条数，`uint16`。
	if len(data) < 2 {
		return fmt.Errorf("invalid 452 response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	// 条目区起始偏移。
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条定长 13 字节，先校验剩余长度防止越界。
		if pos+13 > len(data) {
			return fmt.Errorf("invalid 452 item %d", i)
		}
		// 首字节市场代码；1:5 为`uint32` 编号；5:9、9:13 依次为两个 `float32` 特征值。
		codeNum := binary.LittleEndian.Uint32(data[pos+1 : pos+5])
		item := SecurityFeature452Item{
			Market: data[pos],
			Code:   fmt.Sprintf("%d", codeNum),
			P1:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+5 : pos+9]))),
			P2:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+9 : pos+13]))),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 13
	}
	return nil
}

// Response 返回解码完成的 452 特征查询响应对象。
func (obj *GetSecurityFeature452) Response() *GetSecurityFeature452Reply { return obj.reply }

// ExGetListExtra 表示扩展市场附加列表分页查询协议，返回附加条目列表。
type ExGetListExtra struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetListExtraRequest
	reply      *ExGetListExtraReply
}

// ExGetListExtraRequest 保存附加列表查询的两个待确认参数与条数。
type ExGetListExtraRequest struct {
	A     uint16 // 请求参数一，`uint16`，语义待确认。
	B     uint16 // 请求参数二，`uint16`，语义待确认。
	Count uint16 // 请求条数，`uint16`，传 0 时构造器补默认值 500。
}

// ExGetListExtraReply 保存解码完成的附加列表响应结果。
type ExGetListExtraReply struct {
	Start uint32            // 服务端回显的本页起始偏移，`uint32`。
	Count uint16            // 本页返回条数，`uint16`。
	List  []ExExtraListItem // 解码出的附加条目列表。
}

// ExExtraListItem 表示一条附加列表记录，标志位与数值字段组的确切含义语义待确认。
type ExExtraListItem struct {
	Category uint8    // 市场类别。
	Code     string   // 证券代码，GBK 定长 8 字节解码。
	Flag     uint8    // 标志位，语义待确认。
	Values   []uint16 // 12 个`uint16` 附加数值字段组，语义待确认。
}

// NewExGetListExtra 构造附加列表查询协议实例。
//
// 参数 `req` 指定两个待确认参数与请求条数；传入非空对象时覆盖默认参数，传 `nil` 时使用默认条数
// 500。返回的实例零值不可直接使用。
func NewExGetListExtra(req *ExGetListExtraRequest) *ExGetListExtra {
	obj := &ExGetListExtra{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &ExGetListExtraRequest{Count: 500},
		reply:      new(ExGetListExtraReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXLIST_EXTRA
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetListExtra) applyRequest(req *ExGetListExtraRequest) {
	if req.Count == 0 {
		req.Count = 500
	}
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetListExtra) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXLIST_EXTRA, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码附加列表记录并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；缺少 6 字节头部或条目越界时返回
// 非空错误，解码成功返回 `nil`。
func (obj *ExGetListExtra) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 6 字节：起始偏移 4 字节`uint32`、条数 2 字节`uint16`。
	if len(data) < 6 {
		return fmt.Errorf("invalid 23f6 response length: %d", len(data))
	}
	obj.reply.Start = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Count = binary.LittleEndian.Uint16(data[4:6])
	// 条目区起始偏移。
	pos := 6
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条定长 34 字节，先校验剩余长度防止越界。
		if pos+34 > len(data) {
			return fmt.Errorf("invalid 23f6 item %d", i)
		}
		// 首字节类别，1:9 为 GBK 代码，偏移 9 为标志位。
		item := ExExtraListItem{
			Category: data[pos],
			Code:     Utf8ToGbk(data[pos+1 : pos+9]),
			Flag:     data[pos+9],
		}
		// 连续 12 个`uint16`，步长 2 字节，依次读入 `Values` 数组。
		item.Values = make([]uint16, 12)
		for j := 0; j < 12; j++ {
			item.Values[j] = binary.LittleEndian.Uint16(data[pos+10+j*2 : pos+12+j*2])
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 34
	}
	return nil
}

// Response 返回解码完成的附加列表响应对象。
func (obj *ExGetListExtra) Response() *ExGetListExtraReply { return obj.reply }

// ExExperiment2487 表示 2487 消息号试验性单品种行情查询协议；响应尾部未逆向字节以十六进制文本保留。
type ExExperiment2487 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExExperiment2487Request
	reply      *ExExperiment2487Reply
}

// ExExperiment2487Request 保存 2487 试验性行情查询的市场类别与代码参数。
type ExExperiment2487Request struct {
	Category uint8    // 市场类别。
	Code     [23]byte // 证券代码，GBK 定长 23 字节，不足补零。
}

// ExExperiment2487Reply 保存解码完成的 2487 行情查询响应结果。
type ExExperiment2487Reply struct {
	Category uint8   // 回显市场类别。
	Code     string  // 证券代码，GBK 解码。
	Active   uint32  // 交易状态/活跃标志，`uint32`，语义待确认。
	PreClose float64 // 昨收价，由 `float32` 还原，单位元。
	Open     float64 // 开盘价，由 `float32` 还原，单位元。
	High     float64 // 最高价，由 `float32` 还原，单位元。
	Low      float64 // 最低价，由 `float32` 还原，单位元。
	Close    float64 // 最新价/收盘价，由 `float32` 还原，单位元。
	U1       float64 // 未知行情字段，由 `float32` 还原，语义待确认。
	Price    float64 // 当前价格，由 `float32` 还原，单位元。
	Vol      uint32  // 累计成交量，`uint32`。
	CurVol   uint32  // 当前成交量，`uint32`。
	Amount   float64 // 成交额，由 `float32` 还原，单位元。
	TailHex  string  // 68 字节定长头部之后未逆向字节的小写十六进制文本；无尾部时为空串。
}

// NewExExperiment2487 构造 2487 行情查询协议实例。
//
// 参数 `req` 指定市场类别与证券代码；传入非空对象时覆盖默认参数，传 `nil` 时使用空请求。返回的
// 实例零值不可直接使用。
func NewExExperiment2487(req *ExExperiment2487Request) *ExExperiment2487 {
	obj := &ExExperiment2487{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExExperiment2487Request),
		reply:      new(ExExperiment2487Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES_EXPERIMENT1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExExperiment2487) applyRequest(req *ExExperiment2487Request) { obj.request = req }

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExExperiment2487) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXQUOTES_EXPERIMENT1, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码 2487 行情数据并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；定长头部不足 68 字节时返回非空
// 错误，解码成功返回 `nil`，68 字节之后的剩余字节原样保留到 `TailHex`。
func (obj *ExExperiment2487) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 定长行情头部至少需要 68 字节。
	if len(data) < 68 {
		return fmt.Errorf("invalid 2487 response length: %d", len(data))
	}
	// 偏移 0 为类别，1:24 为 GBK 代码，24:28 为状态`uint32`。
	obj.reply.Category = data[0]
	obj.reply.Code = Utf8ToGbk(data[1:24])
	obj.reply.Active = binary.LittleEndian.Uint32(data[24:28])
	// 28:56 依次为昨收、开盘、最高、最低、收盘、未知字段、当前价格七个 `float32`。
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[28:32])))
	obj.reply.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[32:36])))
	obj.reply.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[36:40])))
	obj.reply.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[40:44])))
	obj.reply.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[44:48])))
	obj.reply.U1 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[48:52])))
	obj.reply.Price = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[52:56])))
	// 56:60、60:64 为成交量与当前成交量`uint32`，64:68 为成交额 `float32`。
	obj.reply.Vol = binary.LittleEndian.Uint32(data[56:60])
	obj.reply.CurVol = binary.LittleEndian.Uint32(data[60:64])
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[64:68])))
	// 68 字节之后为未逆向的协议尾部，保留原始字节的小写十六进制文本。
	if len(data) > 68 {
		obj.reply.TailHex = hex.EncodeToString(data[68:])
	}
	return nil
}

// Response 返回解码完成的 2487 行情响应对象。
func (obj *ExExperiment2487) Response() *ExExperiment2487Reply { return obj.reply }

// ExExperiment2488 表示 2488 消息号试验性多字段数据查询协议，返回记录列表。
type ExExperiment2488 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExExperiment2488Request
	reply      *ExExperiment2488Reply
}

// ExExperiment2488Request 保存 2488 试验性数据查询参数，查询模式默认 55。
type ExExperiment2488Request struct {
	Category uint8    // 市场类别。
	Code     [23]byte // 证券代码，GBK 定长 23 字节，不足补零。
	Zero1    uint32   // 保留置零字段，`uint32`，语义待确认。
	Mode     uint16   // 查询模式，`uint16`，传 0 时构造器补默认值 55。
	Zero2    uint32   // 保留置零字段，`uint32`，语义待确认。
	Zero3    uint32   // 保留置零字段，`uint32`，语义待确认。
}

// ExExperiment2488Reply 保存解码完成的 2488 数据查询响应结果。
type ExExperiment2488Reply struct {
	Category uint8                  // 回显市场类别。
	Code     string                 // 证券代码，GBK 解码。
	Count    uint16                 // 返回条目数，`uint16`。
	List     []ExExperiment2488Item // 解码出的数据条目列表。
}

// ExExperiment2488Item 表示一条 2488 试验性数据记录，数值字段组的确切含义语义待确认。
type ExExperiment2488Item struct {
	ID     uint32   // 条目标识，`uint32`。
	Values []uint16 // 6 个`uint16` 数值字段组，语义待确认。
}

// NewExExperiment2488 构造 2488 数据查询协议实例。
//
// 参数 `req` 指定市场类别、代码、保留字段与查询模式；传入非空对象时覆盖默认参数，传 `nil` 时使用
// 默认模式 55。返回的实例零值不可直接使用。
func NewExExperiment2488(req *ExExperiment2488Request) *ExExperiment2488 {
	obj := &ExExperiment2488{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &ExExperiment2488Request{Mode: 55},
		reply:      new(ExExperiment2488Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES_EXPERIMENT2
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExExperiment2488) applyRequest(req *ExExperiment2488Request) {
	if req.Mode == 0 {
		req.Mode = 55
	}
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExExperiment2488) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXQUOTES_EXPERIMENT2, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码 2488 数据记录并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；头部不足 38 字节或条目越界时
// 返回非空错误，解码成功返回 `nil`。
func (obj *ExExperiment2488) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 38 字节：类别 1、代码 35 字节 GBK、条数 2 字节`uint16`。
	if len(data) < 38 {
		return fmt.Errorf("invalid 2488 response length: %d", len(data))
	}
	obj.reply.Category = data[0]
	obj.reply.Code = Utf8ToGbk(data[1:36])
	obj.reply.Count = binary.LittleEndian.Uint16(data[36:38])
	// 条目区起始偏移。
	pos := 38
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条定长 16 字节，先校验剩余长度防止越界。
		if pos+16 > len(data) {
			return fmt.Errorf("invalid 2488 item %d", i)
		}
		// 前 4 字节为`uint32` 标识，随后六个`uint16` 数值字段。
		item := ExExperiment2488Item{
			ID:     binary.LittleEndian.Uint32(data[pos : pos+4]),
			Values: make([]uint16, 6),
		}
		// 连续六个`uint16`，步长 2 字节。
		for j := 0; j < 6; j++ {
			item.Values[j] = binary.LittleEndian.Uint16(data[pos+4+j*2 : pos+6+j*2])
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 16
	}
	return nil
}

// Response 返回解码完成的 2488 数据响应对象。
func (obj *ExExperiment2488) Response() *ExExperiment2488Reply { return obj.reply }

// ExGetKLine2 表示扩展市场 K 线第二种协议变体；与基础变体不同，本协议不舍弃首根 K 线，
// 也不滚动计算昨收与涨跌。
type ExGetKLine2 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetKLine2Request
	reply      *ExGetKLine2Reply
}

// ExGetKLine2Request 保存 K 线第二变体的市场类别、代码、周期与条数参数。
type ExGetKLine2Request struct {
	Category uint8    // 市场类别。
	Code     [23]byte // 证券代码，GBK 定长 23 字节，不足补零。
	Period   uint16   // K 线周期类别，`uint16`，取值由服务端周期表约定。
	Times    uint16   // 周期倍数，`uint16`，传 0 时构造器补默认值 1。
	Start    uint32   // 起始偏移条数，`uint32`，0 表示从最新一根向前取数。
	Count    uint32   // 请求 K 线条数，`uint32`，传 0 时构造器补默认值 800。
	Reserved [16]byte // 16 字节保留填充区，发送时置零，语义待确认。
}

// ExGetKLine2Reply 保存解码完成的 K 线第二变体响应结果。
type ExGetKLine2Reply struct {
	Category uint8         // 回显市场类别。
	Name     string        // 证券名称，GBK 解码。
	Period   uint16        // 回显 K 线周期类别，`uint16`。
	Times    uint16        // 回显周期倍数，`uint16`。
	Start    uint32        // 服务端回显的起始偏移条数，`uint32`。
	Count    uint16        // 实际返回条数，`uint16`。
	List     []ExKLineItem // 解码出的 K 线列表，按时间顺序排列；本变体不舍弃首根、不计算涨跌。
}

// NewExGetKLine2 构造 K 线第二变体查询协议实例。
//
// 参数 `req` 指定市场类别、代码、周期类别、周期倍数、起始偏移与请求条数；传入非空对象时覆盖默认
// 参数，传 `nil` 时使用空请求，周期倍数为 0 补 1、条数为 0 补 800。返回的实例零值不可直接使用。
func NewExGetKLine2(req *ExGetKLine2Request) *ExGetKLine2 {
	obj := &ExGetKLine2{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetKLine2Request),
		reply:      new(ExGetKLine2Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXKLINE2
	if req != nil {
		// 多请求一根 K 线，作为首根昨收基准，响应时剔除，保证返回条数与请求一致。
		req.Count = req.Count + 1
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetKLine2) applyRequest(req *ExGetKLine2Request) {
	if req.Times == 0 {
		req.Times = 1
	}
	if req.Count == 0 {
		req.Count = 800
	}
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetKLine2) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXKLINE2, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码 K 线第二变体数据并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；头部不足 42 字节、日期编号解码
// 失败或 K 线条目越界时返回非空错误，解码成功返回 `nil`。
func (obj *ExGetKLine2) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 42 字节：类别 1、名称 23 字节 GBK、周期 2、倍数 2、起始偏移 4、填充区与条数 2。
	if len(data) < 42 {
		return fmt.Errorf("invalid ex kline2 response length: %d", len(data))
	}
	obj.reply.Category = data[0]
	obj.reply.Name = Utf8ToGbk(data[1:24])
	obj.reply.Period = binary.LittleEndian.Uint16(data[24:26])
	obj.reply.Times = binary.LittleEndian.Uint16(data[26:28])
	obj.reply.Start = binary.LittleEndian.Uint32(data[28:32])
	obj.reply.Count = binary.LittleEndian.Uint16(data[40:42])
	// K 线条目区起始偏移。
	pos := 42
	// 滚动保存上一根收盘价，作为本根昨收基准。
	var preCloseRaw float64 // 昨收盘价
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每根 K 线步长 32 字节，先校验剩余长度防止越界。
		if pos+32 > len(data) {
			return fmt.Errorf("invalid ex kline2 item %d", i)
		}
		dateNum := binary.LittleEndian.Uint32(data[pos : pos+4])
		// 日期编号按周期类型解码为时间戳，解码失败直接终止解析。
		ts, ok := decodeDateNum(obj.reply.Period, dateNum, false)
		if !ok {
			return fmt.Errorf("invalid ex kline2 datetime: %d", dateNum)
		}
		item := ExKLineItem{
			DateTime: ts,
			// 开、高、低、收、成交额依次为五个 `float32`，成交量为`uint32`。
			Open:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))),
			High:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))),
			Low:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))),
			Close:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+16 : pos+20]))),
			Amount: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+20 : pos+24]))),
			Vol:    binary.LittleEndian.Uint32(data[pos+24 : pos+28]),
		}
		// 本根昨收/前收取上一根收盘价；随后把基准更新为本根收盘，供下一根使用。
		item.PreClose = float64(preCloseRaw)
		preCloseRaw = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+16 : pos+20])))
		item.RiseRate = item.GetRiseRate()
		item.RisePrice = item.GetRisePrice()
		pos += 32
		// 首根仅用于取得昨收基准，不进入最终 K 线列表。
		if i == 0 {
			continue
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	// 首根作为昨收基准已剔除；服务器可能返回 0 条，避免无符号下溢。
	if obj.reply.Count > 0 {
		obj.reply.Count--
	}
	return nil
}

// Response 返回解码完成的 K 线第二变体响应对象。
func (obj *ExGetKLine2) Response() *ExGetKLine2Reply { return obj.reply }

// ExMapping2562 表示 2562 消息号试验性扩展市场代码映射表分页查询协议，
// 除类别、名称与序号外其余字段含义未确认。
type ExMapping2562 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExMapping2562Request
	reply      *ExMapping2562Reply
}

// ExMapping2562Request 保存映射表查询的市场代码、起始偏移与条数参数。
type ExMapping2562Request struct {
	Market uint16 // 市场代码，`uint16`。
	Start  uint32 // 起始偏移条数，`uint32`，0 表示从第一条开始。
	Count  uint32 // 请求条数，`uint32`，传 0 时构造器补默认值 600。
}

// ExMapping2562Reply 保存解码完成的映射表响应结果。
type ExMapping2562Reply struct {
	Count uint16              // 返回条数，`uint16`。
	List  []ExMapping2562Item // 解码出的映射条目列表。
}

// ExMapping2562Item 表示一条 2562 映射记录，除名称与序号外多数字段的确切含义语义待确认。
type ExMapping2562Item struct {
	Category uint16  // 类别代码，`uint16`。
	Name     string  // 名称，GBK 定长 23 字节解码。
	Unknown  uint16  // 未知字段，`uint16`，语义待确认。
	Index    uint32  // 序号/索引，`uint32`。
	Switch   uint8   // 开关标志，语义待确认。
	Code1    float64 // 映射数值一，由 `float32` 还原，语义待确认。
	Code2    float64 // 映射数值二，由 `float32` 还原，语义待确认。
	Code3    float64 // 映射数值三，由 `float32` 还原，语义待确认。
	Code4    uint16  // 映射数值四，`uint16`，语义待确认。
	Code5    uint16  // 映射数值五，`uint16`，语义待确认。
}

// NewExMapping2562 构造映射表查询协议实例。
//
// 参数 `req` 指定市场代码、起始偏移与请求条数；传入非空对象时覆盖默认参数，传 `nil` 时使用默认
// 条数 600。返回的实例零值不可直接使用。
func NewExMapping2562(req *ExMapping2562Request) *ExMapping2562 {
	obj := &ExMapping2562{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &ExMapping2562Request{Count: 600},
		reply:      new(ExMapping2562Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXMAPPING2562
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExMapping2562) applyRequest(req *ExMapping2562Request) {
	if req.Count == 0 {
		req.Count = 600
	}
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExMapping2562) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXMAPPING2562, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码映射记录并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；缺少 2 字节条数头部或映射条目
// 越界时返回非空错误，解码成功返回 `nil`。
func (obj *ExMapping2562) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 2 字节条数，`uint16`。
	if len(data) < 2 {
		return fmt.Errorf("invalid 2562 response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	// 条目区起始偏移。
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条定长 48 字节，先校验剩余长度防止越界。
		if pos+48 > len(data) {
			return fmt.Errorf("invalid 2562 item %d", i)
		}
		// 0:2 类别 `uint16`，2:25 名称 GBK，25:27 未知 `uint16`，27:31 序号 `uint32`，
		// 31 为开关；32:44 三个 `float32`，44:48 两个 `uint16`。
		item := ExMapping2562Item{
			Category: binary.LittleEndian.Uint16(data[pos : pos+2]),
			Name:     Utf8ToGbk(data[pos+2 : pos+25]),
			Unknown:  binary.LittleEndian.Uint16(data[pos+25 : pos+27]),
			Index:    binary.LittleEndian.Uint32(data[pos+27 : pos+31]),
			Switch:   data[pos+31],
			Code1:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+32 : pos+36]))),
			Code2:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+36 : pos+40]))),
			Code3:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+40 : pos+44]))),
			Code4:    binary.LittleEndian.Uint16(data[pos+44 : pos+46]),
			Code5:    binary.LittleEndian.Uint16(data[pos+46 : pos+48]),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 48
	}
	return nil
}

// Response 返回解码完成的映射表响应对象。
func (obj *ExMapping2562) Response() *ExMapping2562Reply { return obj.reply }

// mustDecodeHex 把固定十六进制字面量解码为字节。
//
// 参数 `value` 为源码中写死的十六进制字符串；返回值为解码后的字节切片。字面量在编译期固定，格式
// 非法说明为常量错误，解码失败即直接 `panic` 中止程序。
func mustDecodeHex(value string) []byte {
	out, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return out
}
