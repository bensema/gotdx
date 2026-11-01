package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// ExGetCount 表示扩展市场证券品种总数查询协议，无请求参数体。
type ExGetCount struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExGetCountReply
}

// ExGetCountReply 保存解码完成的品种总数查询响应结果。
type ExGetCountReply struct {
	Count uint32 // 扩展市场证券品种总数，`uint32`，取自响应偏移 19 起的 4 字节。
}

// NewExGetCount 构造品种总数查询协议实例；该协议不接受入参，返回的实例零值不可直接使用。
func NewExGetCount() *ExGetCount {
	obj := &ExGetCount{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExGetCountReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXCOUNT
	return obj
}

// BuildRequest 构造品种总数查询请求字节流。
//
// 该协议无载荷，仅由协议头与消息号组成；返回值第一项为请求字节流，不返回错误。
func (obj *ExGetCount) BuildRequest() ([]byte, error) {
	return buildExRequest(KMSG_EXCOUNT, nil)
}

// ParseResponse 从服务端响应字节流解码品种总数并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应不足 23 字节（19 字节保留区
// 加 4 字节总数）时返回非空错误，解码成功返回 `nil`。
func (obj *ExGetCount) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 至少需要 23 字节（19 字节保留区 + 4 字节总数）。
	if len(data) < 23 {
		return fmt.Errorf("invalid ex count response length: %d", len(data))
	}
	// 前 19 字节为固定保留区，总数位于偏移 19 起 4 字节`uint32`。
	obj.reply.Count = binary.LittleEndian.Uint32(data[19:23])
	return nil
}

// Response 返回解码完成的品种总数响应对象。
func (obj *ExGetCount) Response() *ExGetCountReply {
	return obj.reply
}

// ExGetCategoryList 表示扩展市场品种分类目录查询协议，无请求参数体。
type ExGetCategoryList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExGetCategoryListReply
}

// ExGetCategoryListReply 保存解码完成的分类目录响应结果。
type ExGetCategoryListReply struct {
	Count uint16           // 返回分类条数，`uint16`。
	List  []ExCategoryItem // 解码出的分类条目列表。
}

// ExCategoryItem 表示一个扩展市场分类条目，包含品种类型、市场与名称等信息。
type ExCategoryItem struct {
	GoodsType     uint8  // 品种类型代码，同时作为市场代码使用。
	GoodsTypeName string // 品种类型名称，由类型代码经映射表转换得到。
	Market        uint8  // 所属市场代码，取值与 `GoodsType` 相同。
	Name          string // 分类名称，GBK 定长 32 字节解码。
	Code          uint8  // 分类代码。
	Abbr          string // 分类缩写，GBK 定长 30 字节解码。
}

// NewExGetCategoryList 构造分类目录查询协议实例；该协议不接受入参，返回的实例零值不可直接使用。
func NewExGetCategoryList() *ExGetCategoryList {
	obj := &ExGetCategoryList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExGetCategoryListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXCATEGORYLIST
	return obj
}

// BuildRequest 构造分类目录查询请求字节流。
//
// 该协议无载荷，仅由协议头与消息号组成；返回值第一项为请求字节流，不返回错误。
func (obj *ExGetCategoryList) BuildRequest() ([]byte, error) {
	return buildExRequest(KMSG_EXCATEGORYLIST, nil)
}

// ParseResponse 从服务端响应字节流解码分类目录并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；缺少 2 字节条数头部或分类条目
// 越界时返回非空错误，解码成功返回 `nil`。
func (obj *ExGetCategoryList) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 2 字节条数，`uint16`。
	if len(data) < 2 {
		return fmt.Errorf("invalid ex category list response length: %d", len(data))
	}

	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条定长 64 字节，先校验剩余长度防止越界。
		base := 2 + int(i)*64
		if base+64 > len(data) {
			return fmt.Errorf("invalid ex category list item %d", i)
		}
		goodsType := data[base]
		obj.reply.List = append(obj.reply.List, ExCategoryItem{
			GoodsType:     goodsType,
			GoodsTypeName: exGoodsTypeName(goodsType),
			Market:        goodsType,
			// 名称位于 `base+1` 起 32 字节 GBK；分类代码为 `base+33`；缩写位于 `base+34` 起 30 字节 GBK。
			Name: Utf8ToGbk(data[base+1 : base+33]),
			Code: data[base+33],
			Abbr: Utf8ToGbk(data[base+34 : base+64]),
		})
	}
	return nil
}

// Response 返回解码完成的分类目录响应对象。
func (obj *ExGetCategoryList) Response() *ExGetCategoryListReply {
	return obj.reply
}

// ExGetList 表示扩展市场证券品种分页列表查询协议，返回含简要描述字段的证券条目。
type ExGetList struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetListRequest
	reply      *ExGetListReply
}

// ExGetListRequest 保存证券列表分页查询的起始偏移与条数参数。
type ExGetListRequest struct {
	Start uint32 // 起始偏移条数，`uint32`，0 表示从第一条开始。
	Count uint16 // 本页请求条数，`uint16`。
}

// ExGetListReply 保存解码完成的证券列表响应结果。
type ExGetListReply struct {
	Start uint32       // 服务端回显的本页起始偏移，`uint32`。
	Count uint16       // 本页返回条数，`uint16`。
	List  []ExListItem // 解码出的证券条目列表。
}

// ExListItem 表示列表中的一条扩展市场证券条目，含市场、类别、代码、名称及若干描述字段。
type ExListItem struct {
	Market   uint8     // 市场代码。
	Category uint8     // 品种类别。
	Code     string    // 证券代码，GBK 定长 9 字节解码。
	Name     string    // 证券名称，GBK 定长 26 字节解码。
	Desc     []float64 // 混合类型行情/指标描述数组（12 个字段），具体含义随品种类别变化，语义待确认。
}

// NewExGetList 构造证券列表查询协议实例。
//
// 参数 `req` 指定起始偏移与本页条数；传入非空对象时覆盖默认参数，传 `nil` 时使用空请求。返回的
// 实例零值不可直接使用。
func NewExGetList(req *ExGetListRequest) *ExGetList {
	obj := &ExGetList{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetListRequest),
		reply:      new(ExGetListReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXLIST
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetList) applyRequest(req *ExGetListRequest) {
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetList) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXLIST, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码证券列表并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；缺少 6 字节头部或证券条目越界时
// 返回非空错误，解码成功返回 `nil`。
func (obj *ExGetList) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 头部至少需要 6 字节：起始偏移 4 字节`uint32`、条数 2 字节`uint16`。
	if len(data) < 6 {
		return fmt.Errorf("invalid ex list response length: %d", len(data))
	}

	obj.reply.Start = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Count = binary.LittleEndian.Uint16(data[4:6])
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条定长 64 字节，先校验剩余长度防止越界。
		base := 6 + int(i)*64
		if base+64 > len(data) {
			return fmt.Errorf("invalid ex list item %d", i)
		}
		// Desc 为混合类型描述数组：、`uint16`、 `float32` 交错排布，
		// 字段含义随品种类别变化，语义待确认。
		item := ExListItem{
			Market:   data[base],
			Category: data[base+1],
			Code:     Utf8ToGbk(data[base+5 : base+14]),
			Name:     Utf8ToGbk(data[base+14 : base+40]),
			Desc: []float64{
				float64(data[base+2]),
				float64(binary.LittleEndian.Uint16(data[base+3 : base+5])),
				float64(math.Float32frombits(binary.LittleEndian.Uint32(data[base+40 : base+44]))),
				float64(math.Float32frombits(binary.LittleEndian.Uint32(data[base+44 : base+48]))),
				float64(binary.LittleEndian.Uint16(data[base+48 : base+50])),
				float64(binary.LittleEndian.Uint16(data[base+50 : base+52])),
				float64(binary.LittleEndian.Uint16(data[base+52 : base+54])),
				float64(binary.LittleEndian.Uint16(data[base+54 : base+56])),
				float64(binary.LittleEndian.Uint16(data[base+56 : base+58])),
				float64(binary.LittleEndian.Uint16(data[base+58 : base+60])),
				float64(binary.LittleEndian.Uint16(data[base+60 : base+62])),
				float64(binary.LittleEndian.Uint16(data[base+62 : base+64])),
			},
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	return nil
}

// Response 返回解码完成的证券列表响应对象。
func (obj *ExGetList) Response() *ExGetListReply {
	return obj.reply
}
