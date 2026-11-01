package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// MACBoardListRequest 保存板块总数查询与板块列表查询共用的分页、板块类型与排序参数。
//
// 分页与排序相关字段为 0 时由构造函数回填默认值。
type MACBoardListRequest struct {
	PageSize  uint16  // 每页条数，为 0 时默认取 150。
	BoardType uint16  // 板块类型代码，取值见 `types.BoardType*` 常量。
	SortType  uint8   // 排序字段编号，取值见根包 `MACSortType*` 常量，本对象未为其设置构造默认值。
	SortOrder uint8   // 排序方向编号，取值见根包 `MACSortOrder*` 常量（1 为降序、2 为升序、0 为不指定），为 0 时默认回填 1。
	Start     uint16  // 起始板块序号（分页偏移），两字节。
	One       uint16  // 协议固定标志位，固定填 1，语义待确认。
	Reserved  [8]byte // 协议保留填充字节，填零。
}

// MACBoardCount 查询 MAC 板块总数，返回结果供上层计算分页。
//
// 使用时先由 NewMACBoardCount 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACBoardCount struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardListRequest
	reply      *MACBoardCountReply
}

// MACBoardCountReply 保存板块总数查询返回的两组计数。
type MACBoardCountReply struct {
	CountAll uint16 // 板块与标的成对统计的原始计数，两字节。
	Total    uint16 // 另一组板块计数，两字节，口径语义待确认。
}

// NewMACBoardCount 创建板块总数查询对象；请求参数为空时使用内置默认参数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`BoardType` 为板块类型、`SortType` 为排序字段、
// `SortOrder` 为排序方向、`Start` 为起始序号；`PageSize`、`SortOrder`、`One` 为 0 时回填默认值。
func NewMACBoardCount(req *MACBoardListRequest) *MACBoardCount {
	obj := &MACBoardCount{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardListRequest),
		reply:      new(MACBoardCountReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXBOARDLIST
	obj.request.PageSize = 150
	obj.request.SortOrder = 1
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACBoardCount) applyRequest(req *MACBoardListRequest) {
	if req.PageSize == 0 {
		req.PageSize = 150
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_EXBOARDLIST` 与请求头封装成完整报文。
func (obj *MACBoardCount) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_EXBOARDLIST, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACBoardCount) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 4 {
		return fmt.Errorf("invalid mac board count response length: %d", len(data))
	}
	// 两组计数均依次读取。
	obj.reply.CountAll = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Total = binary.LittleEndian.Uint16(data[2:4])
	return nil
}

// Response 返回解码完成的总数响应；未调用 ParseResponse 前内容为零值。
func (obj *MACBoardCount) Response() *MACBoardCountReply {
	return obj.reply
}

// MACBoardList 分页查询 MAC 板块列表，同时返回板块自身行情与代表标的行情。
//
// 使用时先由 NewMACBoardList 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACBoardList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardListRequest
	reply      *MACBoardListReply
}

// MACBoardListReply 保存板块列表查询返回的计数与逐条板块条目。
type MACBoardListReply struct {
	CountAll uint16             // 板块与标的成对统计的原始计数，两字节。
	Total    uint16             // 另一组板块计数，两字节，口径语义待确认。
	Count    uint16             // 折算后的板块条目数，供遍历 `List` 使用。
	List     []MACBoardListItem // 逐条板块条目，按排序结果顺序排列。
}

// MACBoardListItem 表示一个板块条目；无前缀字段为板块自身行情，
// 带代表标的前缀的字段为该板块代表标的的行情快照。
type MACBoardListItem struct {
	Market          uint16  // 板块市场代码，两字节。
	Code            string  // 板块代码，GBK 解码。
	Name            string  // 板块名称，GBK 解码。
	Price           float64 // 板块最新价，单精度浮点还原。
	RiseSpeed       float64 // 板块涨速，单精度浮点还原。
	PreClose        float64 // 板块昨收价，单精度浮点还原。
	SymbolMarket    uint16  // 代表标的市场代码，两字节。
	SymbolCode      string  // 代表标的代码，GBK 解码。
	SymbolName      string  // 代表标的名称，GBK 解码。
	SymbolPrice     float64 // 代表标的最新价，单精度浮点还原。
	SymbolRiseSpeed float64 // 代表标的涨速，单精度浮点还原。
	SymbolPreClose  float64 // 代表标的昨收价，单精度浮点还原。
}

// NewMACBoardList 创建板块列表查询对象；请求参数为空时使用内置默认参数。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`BoardType` 为板块类型、`SortType` 为排序字段、
// `SortOrder` 为排序方向、`Start` 为起始序号；`PageSize`、`SortOrder`、`One` 为 0 时回填默认值。
func NewMACBoardList(req *MACBoardListRequest) *MACBoardList {
	obj := &MACBoardList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardListRequest),
		reply:      new(MACBoardListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXBOARDLIST
	obj.request.PageSize = 150
	obj.request.SortOrder = 1
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACBoardList) applyRequest(req *MACBoardListRequest) {
	if req.PageSize == 0 {
		req.PageSize = 150
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_EXBOARDLIST` 与请求头封装成完整报文。
func (obj *MACBoardList) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_EXBOARDLIST, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或记录区不完整时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACBoardList) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 4 {
		return fmt.Errorf("invalid mac board list response length: %d", len(data))
	}

	// 计数头两组计数均读取。
	obj.reply.CountAll = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Total = binary.LittleEndian.Uint16(data[2:4])
	// 计数通常按板块与标的成对统计，条目数先除以 2；折算为 0 而原始计数非 0 时回退，
	// 用于兼容服务端返回单条布局的情形。
	obj.reply.Count = obj.reply.CountAll / 2
	if obj.reply.Count == 0 && obj.reply.CountAll > 0 {
		obj.reply.Count = obj.reply.CountAll
	}

	// 记录区自偏移 4 开始，每条定长 160 字节。
	pos := 4
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 剩余字节不足一条记录时中止并报错，防止越界读取。
		if pos+160 > len(data) {
			return fmt.Errorf("invalid mac board list item %d", i)
		}
		// 记录分两段：前半为板块自身行情，后半为代表标的行情；
		// 价格、涨速、昨收均为单精度浮点。
		item := MACBoardListItem{
			Market:          binary.LittleEndian.Uint16(data[pos : pos+2]),
			Code:            Utf8ToGbk(data[pos+2 : pos+8]),
			Name:            Utf8ToGbk(data[pos+24 : pos+68]),
			Price:           float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+68 : pos+72]))),
			RiseSpeed:       float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+72 : pos+76]))),
			PreClose:        float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+76 : pos+80]))),
			SymbolMarket:    binary.LittleEndian.Uint16(data[pos+80 : pos+82]),
			SymbolCode:      Utf8ToGbk(data[pos+82 : pos+88]),
			SymbolName:      Utf8ToGbk(data[pos+104 : pos+148]),
			SymbolPrice:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+148 : pos+152]))),
			SymbolRiseSpeed: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+152 : pos+156]))),
			SymbolPreClose:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+156 : pos+160]))),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 160
	}

	return nil
}

// Response 返回解码完成的板块列表响应；未调用 ParseResponse 前内容为零值。
func (obj *MACBoardList) Response() *MACBoardListReply {
	return obj.reply
}
