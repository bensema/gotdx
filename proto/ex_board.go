package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// ExGetBoardList 表示扩展市场板块列表查询协议，串联请求构造、报文编码、响应解码与结果获取。
//
// 该实例必须通过 NewExGetBoardList 构造，零值不可用；实例字段应在构造阶段确定，
// 连接建立后不应再修改。
type ExGetBoardList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetBoardListRequest
	reply      *ExGetBoardListReply
}

// ExGetBoardListRequest 保存扩展市场板块列表的分页与排序参数；零值字段由构造函数补默认值。
type ExGetBoardListRequest struct {
	PageSize  uint16  // 每页条数，`uint16`，传 0 时构造器补默认值 300。
	BoardType uint16  // 板块类型，`uint16`，取值由服务端分类约定。
	SortType  uint8   // 排序字段编号，编号表语义待确认。
	SortOrder uint8   // 排序方向，传 0 时构造器补默认值 1；取值与正倒序的对应关系语义待确认。
	Start     uint16  // 起始偏移条数，`uint16`，0 表示从第一条开始。
	One       uint16  // 协议保留开关，`uint16`，传 0 时构造器补默认值 1，语义待确认。
	Reserved  [8]byte // 8 字节保留填充区，发送时置零，语义待确认。
}

// ExGetBoardListReply 保存解码完成的板块列表响应结果。
type ExGetBoardListReply struct {
	Unknown uint16            // 未知头部字段，`uint16`，语义待确认。
	Count   uint16            // 返回条数，`uint16`。
	List    []ExBoardListItem // 解码出的板块条目列表。
}

// ExBoardListItem 表示一条板块条目，同时携带板块自身行情与该板块代表个股行情。
type ExBoardListItem struct {
	Market          uint16  // 板块所属市场代码，`uint16`。
	Code            string  // 板块代码，GBK 定长 22 字节解码。
	Name            string  // 板块名称，GBK 定长 44 字节解码。
	Price           float64 // 板块当前价格，由 `float32` 还原，单位元。
	RiseSpeed       float64 // 板块涨速，由 `float32` 还原。
	PreClose        float64 // 板块昨收价，由 `float32` 还原，单位元。
	SymbolMarket    uint16  // 代表个股所属市场代码，`uint16`。
	SymbolCode      string  // 代表个股代码，GBK 定长 22 字节解码。
	SymbolName      string  // 代表个股名称，GBK 定长 44 字节解码。
	SymbolPrice     float64 // 代表个股当前价格，由 `float32` 还原，单位元。
	SymbolRiseSpeed float64 // 代表个股涨速，由 `float32` 还原。
	SymbolPreClose  float64 // 代表个股昨收价，由 `float32` 还原，单位元。
}

// NewExGetBoardList 构造板块列表查询协议实例。
//
// 参数 `req` 指定每页条数、板块类型、排序字段、排序方向、起始偏移与保留开关；传入非空对象时覆盖
// 默认参数，传 `nil` 时使用默认分页（每页 300 条、正序排列）。返回的实例零值不可直接使用。
func NewExGetBoardList(req *ExGetBoardListRequest) *ExGetBoardList {
	obj := &ExGetBoardList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetBoardListRequest),
		reply:      new(ExGetBoardListReply),
	}
	// 扩展协议固定报文头：压缩标志、递增序列号、报文类型与板块列表消息号。
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXBOARDLIST
	obj.request.PageSize = 300
	obj.request.SortOrder = 1
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetBoardList) applyRequest(req *ExGetBoardListRequest) {
	if req.PageSize == 0 {
		req.PageSize = 300
	}
	if req.One == 0 {
		req.One = 1
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetBoardList) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXBOARDLIST, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码板块列表并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应头部不足 4 字节或板块条目
// 越界时返回非空错误，解码成功返回 `nil`。
func (obj *ExGetBoardList) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少 4 字节：未知字段 2 字节、条目数 2 字节。
	if len(data) < 4 {
		return fmt.Errorf("invalid ex board list response length: %d", len(data))
	}

	obj.reply.Unknown = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Count = binary.LittleEndian.Uint16(data[2:4])
	// 条目区起始偏移。
	pos := 4

	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条定长 160 字节，先校验剩余长度防止越界读取。
		if pos+160 > len(data) {
			return fmt.Errorf("invalid ex board item %d", i)
		}
		// 板块代码、名称分别位于 `pos+2` 起 22 字节、`pos+24` 起 44 字节；
		// 代表个股代码、名称位于 `pos+82` 起 22 字节、`pos+104` 起 44 字节。
		item := ExBoardListItem{
			Market:          binary.LittleEndian.Uint16(data[pos : pos+2]),
			Code:            Utf8ToGbk(data[pos+2 : pos+24]),
			Name:            Utf8ToGbk(data[pos+24 : pos+68]),
			Price:           float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+68 : pos+72]))),
			RiseSpeed:       float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+72 : pos+76]))),
			PreClose:        float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+76 : pos+80]))),
			SymbolMarket:    binary.LittleEndian.Uint16(data[pos+80 : pos+82]),
			SymbolCode:      Utf8ToGbk(data[pos+82 : pos+104]),
			SymbolName:      Utf8ToGbk(data[pos+104 : pos+148]),
			SymbolPrice:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+148 : pos+152]))),
			SymbolRiseSpeed: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+152 : pos+156]))),
			SymbolPreClose:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+156 : pos+160]))),
		}
		obj.reply.List = append(obj.reply.List, item)
		// 按定长步长推进到下一条目。
		pos += 160
	}
	return nil
}

// Response 返回解析完成后的板块列表响应数据。
func (obj *ExGetBoardList) Response() *ExGetBoardListReply {
	return obj.reply
}
