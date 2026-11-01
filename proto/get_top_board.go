package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// GetTopBoard 表示一次获取主行情多维度排行榜协议的完整流程对象。
//
// 排行榜覆盖涨幅、跌幅、振幅、涨速、跌速、量比、正委比、负委比、换手等维度。
// 零值不可直接使用，必须通过 NewGetTopBoard 构造；对象绑定单次查询流程，
// 不支持多协程并发复用。
type GetTopBoard struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetTopBoardRequest
	reply      *GetTopBoardReply
}

// GetTopBoardRequest 是 GetTopBoard 排行榜查询使用的请求参数。
type GetTopBoardRequest struct {
	Category uint8   // 板块或市场分类，1 字节无符号整数。
	Mode     uint8   // 协议模式位，1 字节无符号整数；为 0 时默认取 5。
	Reserved [7]byte // 7 字节定长协议保留区，按预置字节填充。
	Size     uint8   // 每个榜单返回的条目数，1 字节无符号整数；为 0 时默认取 20。
}

// GetTopBoardReply 是 GetTopBoard 排行榜查询解析完成后的响应数据，按榜单维度分组存放条目。
type GetTopBoardReply struct {
	Size               uint8          // 每个榜单的条目数，1 字节无符号整数。
	Increase           []TopBoardItem // 涨幅榜条目，按涨幅由高到低排列。
	Decrease           []TopBoardItem // 跌幅榜条目，按跌幅由高到低排列。
	Amplitude          []TopBoardItem // 振幅榜条目。
	RiseSpeed          []TopBoardItem // 涨速榜条目。
	FallSpeed          []TopBoardItem // 跌速榜条目。
	VolRatio           []TopBoardItem // 量比榜条目。
	PosCommissionRatio []TopBoardItem // 正委比榜，即买盘占优榜单。
	NegCommissionRatio []TopBoardItem // 负委比榜，即卖盘占优榜单。
	Turnover           []TopBoardItem // 换手率榜条目。
}

// TopBoardItem 表示排行榜中的一只证券及其上榜指标值。
type TopBoardItem struct {
	Market uint8   // 市场代码，1 字节无符号整数。
	Code   string  // 6 字节定长证券代码，按 `GBK` 还原。
	Price  float64 // 当前价，单位为元，4 字节的单精度浮点数。
	Value  float64 // 上榜指标值，4 字节的单精度浮点数；口径随榜单维度变化，语义待确认。
}

// NewGetTopBoard 构造排行榜查询对象。
//
// 参数 `req` 指定分类 `Category`、模式位 `Mode` 与每榜条目数 `Size`；`Mode` 为 0
// 时默认 5、`Size` 为 0 时默认 20；`req` 为 `nil` 时使用空请求参数。返回的对象
// 可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetTopBoard(req *GetTopBoardRequest) *GetTopBoard {
	obj := new(GetTopBoard)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetTopBoardRequest)
	obj.reply = new(GetTopBoardReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_TOPBOARD
	obj.request.Mode = 5
	copy(obj.request.Reserved[:], []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x00})
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetTopBoard) applyRequest(req *GetTopBoardRequest) {
	// 模式位缺失时默认 5，每榜条目数缺失时默认 20。
	if req.Mode == 0 {
		req.Mode = 5
	}
	if req.Size == 0 {
		req.Size = 20
	}
	// 保留区未填时沿用构造时预置的默认字节。
	if req.Reserved == [7]byte{} {
		req.Reserved = obj.request.Reserved
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文。
//
// 返回的字节切片可直接写入连接；`PkgLen1` 与 `PkgLen2` 双写为同一负载长度，
// 缓冲区写入失败时返回错误。
func (obj *GetTopBoard) BuildRequest() ([]byte, error) {
	// 请求参数固定 12 字节（分类 1 + 模式 1 + 保留 7 + 条数 1 + 对齐填充），包头长度双写为 0x0c。
	obj.reqHeader.PkgLen1 = 0x0c
	obj.reqHeader.PkgLen2 = 0x0c

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 按榜单顺序解析服务端返回的排行榜报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；记录越界时返回
// `io.ErrUnexpectedEOF`。
func (obj *GetTopBoard) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 首字节：每个榜单返回的条目数。
	if err := binary.Read(bytes.NewBuffer(data[:1]), binary.LittleEndian, &obj.reply.Size); err != nil {
		return err
	}

	pos := 1
	// 九个榜单指针，顺序必须与服务端报文输出顺序严格一致。
	targets := []*[]TopBoardItem{
		&obj.reply.Increase,
		&obj.reply.Decrease,
		&obj.reply.Amplitude,
		&obj.reply.RiseSpeed,
		&obj.reply.FallSpeed,
		&obj.reply.VolRatio,
		&obj.reply.PosCommissionRatio,
		&obj.reply.NegCommissionRatio,
		&obj.reply.Turnover,
	}

	for _, target := range targets {
		for i := uint8(0); i < obj.reply.Size; i++ {
			// 单条记录固定 15 字节，越界即返回报文意外结束错误。
			if pos+15 > len(data) {
				return io.ErrUnexpectedEOF
			}
			// 价格与指标值均为 4 字节的单精度浮点数。
			item := TopBoardItem{
				Market: data[pos],
				Code:   Utf8ToGbk(data[pos+1 : pos+7]),
				Price:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+7 : pos+11]))),
				Value:  float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+11 : pos+15]))),
			}
			*target = append(*target, item)
			pos += 15
		}
	}

	return nil
}

// Response 返回已解析完成的排行榜响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetTopBoard) Response() *GetTopBoardReply {
	return obj.reply
}
