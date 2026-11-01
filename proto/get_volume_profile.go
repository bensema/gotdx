package proto

import (
	"bytes"
	"encoding/binary"
	"math"
)

// GetVolumeProfile 表示一次获取个股成交分布与盘口快照协议的完整流程对象。
//
// 成交分布返回各价位上的成交量分布明细，响应前段附带一份与盘口协议同构的快照。
// 零值不可直接使用，必须通过 NewGetVolumeProfile 构造；对象绑定单次查询流程，
// 不支持多协程并发复用。
type GetVolumeProfile struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetVolumeProfileRequest
	reply      *GetVolumeProfileReply
}

// GetVolumeProfileRequest 是 GetVolumeProfile 成交分布查询使用的请求参数。
type GetVolumeProfileRequest struct {
	Market uint16  // 市场代码，`uint16`。
	Code   [6]byte // 6 字节定长证券代码。
}

// GetVolumeProfileReply 是 GetVolumeProfile 成交分布查询解析完成后的响应数据，包含盘口快照与各价位成交分布档。
type GetVolumeProfileReply struct {
	Count       uint16              // 成交分布档位数，2 字节`uint16`。
	Market      uint8               // 市场代码，1 字节无符号整数。
	Code        string              // 6 字节定长证券代码，按 `GBK` 还原。
	Active      uint16              // 活跃度字段，2 字节`uint16`，语义待确认。
	Close       float64             // 最新价，单位为元，由基准价直接还原。
	Open        float64             // 今开价，单位为元，基准价加差值后除以 100。
	High        float64             // 最高价，单位为元，基准价加差值后除以 100。
	Low         float64             // 最低价，单位为元，基准价加差值后除以 100。
	PreClose    float64             // 昨收价，单位为元，基准价加差值后除以 100。
	ServerTime  string              // 服务端时间，由 `formatServerTime` 格式化。
	NegPrice    float64             // 特殊价格字段，基准差值除以 100，语义待确认。
	Vol         int                 // 总成交量，变长整数解码。
	CurVol      int                 // 现量（最近一笔成交量），变长整数解码。
	Amount      float64             // 总成交额，4 字节的单精度浮点数。
	InVol       int                 // 内盘量（主动卖盘成交量），变长整数解码。
	OutVol      int                 // 外盘量（主动买盘成交量），变长整数解码。
	SAmount     int                 // 上涨家数或卖出相关统计字段，变长整数解码，语义待确认。
	OpenAmount  int                 // 开盘金额，变长整数解码。
	Turnover    float64             // 换手率，由高层接口尽力估算补齐，协议本身不下发。
	BidLevels   []Level             // 买盘三档，按买一至买三顺序排列。
	AskLevels   []Level             // 卖盘三档，按卖一至卖三顺序排列。
	Unknown     uint16              // 尾部未知字段，2 字节`uint16`，语义待确认。
	VolProfiles []VolumeProfileItem // 成交分布明细列表，长度与 `Count` 一致。
}

// VolumeProfileItem 表示成交分布中的一个价格档位，包含该档总量与主买、主卖量。
type VolumeProfileItem struct {
	Price float64 // 档位价格，单位为元，由相邻价位差值累加后除以 100。
	Vol   int     // 该档总成交量，变长整数解码。
	Buy   int     // 主买量，变长整数解码。
	Sell  int     // 主卖量，变长整数解码。
}

// NewGetVolumeProfile 构造成交分布查询对象。
//
// 参数 `req` 指定市场 `Market` 与证券代码 `Code`；`req` 为 `nil` 时使用空请求参数。
// 返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetVolumeProfile(req *GetVolumeProfileRequest) *GetVolumeProfile {
	obj := new(GetVolumeProfile)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetVolumeProfileRequest)
	obj.reply = new(GetVolumeProfileReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_VOLUMEPROFILE
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetVolumeProfile) applyRequest(req *GetVolumeProfileRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为完整协议报文。
//
// 返回的字节切片可直接写入连接；`PkgLen1` 与 `PkgLen2` 双写为同一负载长度，
// 缓冲区写入失败时返回错误。
func (obj *GetVolumeProfile) BuildRequest() ([]byte, error) {
	// 负载固定 10 字节（市场 `uint16` 占 2、代码 6、保留 2），包头长度双写为 0x0a。
	obj.reqHeader.PkgLen1 = 0x0a
	obj.reqHeader.PkgLen2 = 0x0a

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的盘口快照与成交分布档位报文并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节；解码失败时返回错误。
func (obj *GetVolumeProfile) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 快照头部为定长字段，条数、市场、代码、活跃度的偏移依次固定为 0、2、3、9。
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Market = data[2]
	obj.reply.Code = Utf8ToGbk(data[3:9])
	obj.reply.Active = binary.LittleEndian.Uint16(data[9:11])
	pos := 11

	// 基准价与昨收、今开、最高、最低差值；后续价位为基准价加差值。
	basePrice := getprice(data, &pos)
	preCloseDiff := getprice(data, &pos)
	openDiff := getprice(data, &pos)
	highDiff := getprice(data, &pos)
	lowDiff := getprice(data, &pos)
	// 服务端时间原始编码，交 `formatServerTime` 格式化。
	serverTimeRaw := getprice(data, &pos)
	// 特殊价格差值，语义待确认。
	negPriceRaw := getprice(data, &pos)
	// 总成交量与现量，均为变长整数解码。
	obj.reply.Vol = getprice(data, &pos)
	obj.reply.CurVol = getprice(data, &pos)
	// 成交额为 4 字节的单精度浮点数。
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos : pos+4])))
	pos += 4
	// 内盘、外盘与两个统计字段，均为变长整数解码。
	obj.reply.InVol = getprice(data, &pos)
	obj.reply.OutVol = getprice(data, &pos)
	obj.reply.SAmount = getprice(data, &pos)
	obj.reply.OpenAmount = getprice(data, &pos)

	// 三档买卖盘：买/卖价为基准价加档位差值，档位量独立读取。
	for i := 0; i < 3; i++ {
		bid := getprice(data, &pos) + basePrice
		ask := getprice(data, &pos) + basePrice
		bidVol := getprice(data, &pos)
		askVol := getprice(data, &pos)
		obj.reply.BidLevels = append(obj.reply.BidLevels, Level{Price: float64(bid) / 100.0, Vol: bidVol})
		obj.reply.AskLevels = append(obj.reply.AskLevels, Level{Price: float64(ask) / 100.0, Vol: askVol})
	}

	obj.reply.Unknown = binary.LittleEndian.Uint16(data[pos : pos+2])
	pos += 2

	// 分布明细：累计价位逐条叠加差值得到当前档位绝对价位。
	profilePrice := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 价位差值可能以 32 位补码变长编码返回，先做溢出归一化再累加。
		priceDelta := decodeVolumeProfilePriceDelta(getprice(data, &pos))
		vol := getprice(data, &pos)
		buy := getprice(data, &pos)
		sell := getprice(data, &pos)
		profilePrice += priceDelta
		obj.reply.VolProfiles = append(obj.reply.VolProfiles, VolumeProfileItem{
			Price: float64(profilePrice) / 100.0,
			Vol:   vol,
			Buy:   buy,
			Sell:  sell,
		})
	}

	// 快照价位统一以基准价加差值还原，除以 100 得到元。
	obj.reply.Close = float64(basePrice) / 100.0
	obj.reply.Open = float64(basePrice+openDiff) / 100.0
	obj.reply.High = float64(basePrice+highDiff) / 100.0
	obj.reply.Low = float64(basePrice+lowDiff) / 100.0
	obj.reply.PreClose = float64(basePrice+preCloseDiff) / 100.0
	obj.reply.ServerTime = formatServerTime(serverTimeRaw)
	obj.reply.NegPrice = float64(negPriceRaw) / 100.0

	return nil
}

// Response 返回已解析完成的成交分布响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetVolumeProfile) Response() *GetVolumeProfileReply {
	return obj.reply
}

// decodeVolumeProfilePriceDelta 归一化成交分布的价位差值。
//
// 参数 `value` 为 `getprice` 解析出的原始差值，返回归一化后的有符号价位差值。
// 变长编码按有符号整数解析，但分布档位的负差值可能以 32 位补码形式返回，高位被置 1
// 导致结果溢出为大正数；当数值达到或超过 2 的 31 次方时，先转回无符号 32 位整数再按
// 有符号 32 位整数解释，把溢出值还原为正确的负差值，其余数值原样返回。
func decodeVolumeProfilePriceDelta(value int) int {
	// 高位被置 1：按 32 位补码重新解释为有符号整数。
	if value >= 1<<31 {
		return int(int32(uint32(value)))
	}
	return value
}
