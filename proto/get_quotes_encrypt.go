package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// GetQuotesEncrypt 表示一次批量加密行情快照查询协议的完整流程对象。
//
// 该对象持有请求头、请求参数与响应数据；零值不可直接使用，必须通过
// NewGetQuotesEncrypt 构造。对象绑定单次查询流程，不支持多协程并发复用。
// 服务端响应经过异或加密，由 `ParseResponse` 在解析前统一解密。
type GetQuotesEncrypt struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetQuotesEncryptRequest
	reply      *GetQuotesEncryptReply
}

// GetQuotesEncryptRequest 是 GetQuotesEncrypt 查询使用的请求参数。
type GetQuotesEncryptRequest struct {
	Stocks []Stock // 待查询证券列表，每项由市场 `Market` 与代码 `Code` 唯一标识。
}

// GetQuotesEncryptReply 是 GetQuotesEncrypt 查询解析完成后的响应数据。
type GetQuotesEncryptReply struct {
	Count uint16               // 返回快照条数，`uint16`。
	List  []EncryptedQuoteItem // 解密后的行情快照列表，长度与 `Count` 一致。
}

// EncryptedQuoteItem 表示一条解密后的加密行情快照，包含价格、量额与买卖档位。
type EncryptedQuoteItem struct {
	Market     uint8   // 市场代码，1 字节无符号整数。
	Code       string  // 6 字节定长证券代码，按 `GBK` 还原。
	Active     uint16  // 活跃度字段，2 字节`uint16`，语义待确认。
	Close      float64 // 最新价，单位为元，由基准价直接还原。
	PreClose   float64 // 昨收价，单位为元，基准价加差值后除以 100。
	Open       float64 // 今开价，单位为元，基准价加差值后除以 100。
	High       float64 // 最高价，单位为元，基准价加差值后除以 100。
	Low        float64 // 最低价，单位为元，基准价加差值后除以 100。
	Time       string  // 行情时间，格式为时:分:秒，由 4 字节`uint32` 拆分。
	Vol        int     // 总成交量，变长整数解码。
	CurVol     int     // 现量（最近一笔成交量），变长整数解码。
	Amount     float64 // 总成交额，4 字节的单精度浮点数。
	InVol      int     // 内盘量（主动卖盘成交量），变长整数解码。
	OutVol     int     // 外盘量（主动买盘成交量），变长整数解码。
	SAmount    int     // 统计字段，变长整数解码，语义待确认。
	OpenAmount int     // 开盘金额，变长整数解码，语义待确认。
	BidLevels  []Level // 买盘五档列表，按买一至买五顺序排列。
	AskLevels  []Level // 卖盘五档列表，按卖一至卖五顺序排列。
}

// NewGetQuotesEncrypt 构造批量加密行情查询对象。
//
// 参数 `req` 指定待查询证券列表 `Stocks`；`req` 为 `nil` 时使用空请求参数。
// 返回的对象可直接调用 `BuildRequest` 与 `ParseResponse`。
func NewGetQuotesEncrypt(req *GetQuotesEncryptRequest) *GetQuotesEncrypt {
	obj := &GetQuotesEncrypt{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(GetQuotesEncryptRequest),
		reply:      new(GetQuotesEncryptReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_QUOTESENCRYPT
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetQuotesEncrypt) applyRequest(req *GetQuotesEncryptRequest) {
	obj.request = req
}

// BuildRequest 将待查询证券列表编码为协议报文，交由通用请求封装补齐命令字与包头。
//
// 负载先写证券个数，再逐只写入市场、6 字节代码与固定附加标识；返回的字节切片
// 可直接写入连接，缓冲区写入失败时返回错误。
func (obj *GetQuotesEncrypt) BuildRequest() ([]byte, error) {
	buf := new(bytes.Buffer)
	// 先写证券个数。
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(obj.request.Stocks))); err != nil {
		return nil, err
	}
	for _, stock := range obj.request.Stocks {
		code := make([]byte, 6)
		copy(code, stock.Code)
		// 市场 1 字节，随后是补齐到 6 字节的定长代码。
		if err := binary.Write(buf, binary.LittleEndian, stock.Market); err != nil {
			return nil, err
		}
		if _, err := buf.Write(code); err != nil {
			return nil, err
		}
		// 协议固定附加标识字 22234 与档位个数 2，按协议要求原样写入。
		if err := binary.Write(buf, binary.LittleEndian, uint16(22234)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
			return nil, err
		}
	}
	return buildGenericRequest(0x0c, 0, 0x01, KMSG_QUOTESENCRYPT, buf.Bytes())
}

// ParseResponse 先对响应报文做异或解密，再逐条解析行情快照并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为加密响应负载字节；报文长度不足、字段
// 越界或解码失败时返回错误。
func (obj *GetQuotesEncrypt) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 整段报文与 0x93 逐字节异或解密。
	xor := make([]byte, len(data))
	for i := range data {
		xor[i] = data[i] ^ 0x93
	}
	// 至少需要容纳 2 字节条数，否则视为非法报文。
	if len(xor) < 2 {
		return fmt.Errorf("invalid encrypted quotes response length: %d", len(xor))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(xor[:2])
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		// 定长前缀 9 字节越界防护。
		if pos+9 > len(xor) {
			return fmt.Errorf("invalid encrypted quote item %d", i)
		}
		item := EncryptedQuoteItem{
			Market: xor[pos],
			Code:   Utf8ToGbk(xor[pos+1 : pos+7]),
			Active: binary.LittleEndian.Uint16(xor[pos+7 : pos+9]),
		}
		pos += 9
		// `closeRaw` 为基准价，其余价格字段均为相对它的差值。
		closeRaw := getprice(xor, &pos)
		preCloseDiff := getprice(xor, &pos)
		openDiff := getprice(xor, &pos)
		highDiff := getprice(xor, &pos)
		lowDiff := getprice(xor, &pos)
		// 4 字节时间编码，格式为时:分:秒。
		if pos+4 > len(xor) {
			return fmt.Errorf("invalid encrypted quote time %d", i)
		}
		timeRaw := binary.LittleEndian.Uint32(xor[pos : pos+4])
		pos += 4
		// 协议保留价格字段，读取后丢弃。
		_ = getprice(xor, &pos)
		item.Vol = getprice(xor, &pos)
		item.CurVol = getprice(xor, &pos)
		// 成交额为 4 字节的单精度浮点数。
		item.Amount = getfloat32(xor, &pos)
		item.InVol = getprice(xor, &pos)
		item.OutVol = getprice(xor, &pos)
		item.SAmount = getprice(xor, &pos)
		item.OpenAmount = getprice(xor, &pos)
		// 五档买卖盘：买/卖价为基准价加档位差值，档位量为独立变长编码。
		for j := 0; j < 5; j++ {
			bid := getprice(xor, &pos) + closeRaw
			ask := getprice(xor, &pos) + closeRaw
			bidVol := getprice(xor, &pos)
			askVol := getprice(xor, &pos)
			item.BidLevels = append(item.BidLevels, Level{Price: float64(bid) / 100.0, Vol: bidVol})
			item.AskLevels = append(item.AskLevels, Level{Price: float64(ask) / 100.0, Vol: askVol})
		}
		// 10 字节尾部保留区越界防护。
		if pos+10 > len(xor) {
			return fmt.Errorf("invalid encrypted quote tail %d", i)
		}
		pos += 10
		// 尾部 6 组共 24 个协议未使用的变长价格，全部跳过以推进 `pos`。
		for j := 0; j < 6; j++ {
			_ = getprice(xor, &pos)
			_ = getprice(xor, &pos)
			_ = getprice(xor, &pos)
			_ = getprice(xor, &pos)
		}

		// 基准价加差值还原各价位，统一除以 100 得到元。
		item.Close = float64(closeRaw) / 100.0
		item.PreClose = float64(closeRaw+preCloseDiff) / 100.0
		item.Open = float64(closeRaw+openDiff) / 100.0
		item.High = float64(closeRaw+highDiff) / 100.0
		item.Low = float64(closeRaw+lowDiff) / 100.0
		item.Time = fmt.Sprintf("%02d:%02d:%02d", timeRaw/10000, (timeRaw/100)%100, timeRaw%100)
		obj.reply.List = append(obj.reply.List, item)
	}
	return nil
}

// Response 返回已解析完成的加密行情响应数据；未调用 `ParseResponse` 时返回空响应对象。
func (obj *GetQuotesEncrypt) Response() *GetQuotesEncryptReply {
	return obj.reply
}
